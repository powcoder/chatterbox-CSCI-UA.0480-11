https://powcoder.com
代写代考加微信 powcoder
Assignment Project Exam Help
Add WeChat powcoder
// Automated test code for the Chatter protocol.
//
// SECURITY WARNING: This code is meant for educational purposes and may
// contain vulnerabilities or other bugs. Please do not use it for
// security-critical applications.
//
// GRADING NOTES: your code will be evaluate on test cases that are similar,
// but not identical to those here. Note that no test code is provided to
// test that your implementation properly zeroizes old keys. You are free
// to modify this code to implement additional tests or modify the tests,
// your test code will not be graded.
//
// Original version
// Joseph Bonneau February 2019

package chatterbox

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"testing"
)

// parameters for the extended tests
const EXTENDED_TEST_ROUNDS = 10000
const EXTENDED_TEST_PARTICIPANTS = 5

// Rate of messages which will be delivered with modifications
// Set this to non-zero for test of error recovery
const EXTENDED_TEST_ERROR_RATE = 0.2

// number of bytes of fingerprint to display in output
const HANDLE_LENGTH = 4

// turn on to print lots of debugging info
const VERBOSE = false

func SkipOnError(t *testing.T, err error) {
	if err != nil {
		t.Skip(err.Error())
	}
}

func FailOnError(t *testing.T, err error) {
	if err != nil {
		t.Fatal(err.Error())
	}
}

// CheckTestVector checks the a value matches an expeted test vector.
// If it does not, fixed randomness mode is set to false and the test fails.
func CheckTestVector(t *testing.T, value []byte, expectedHex, label string) {
	expected, _ := hex.DecodeString(expectedHex)
	if !bytes.Equal(value, expected) {
		t.Logf("%s did not match expected test vector", label)
		t.Logf("Expected: %0X", expected)
		t.Logf("Got: %0X", value)
		t.Fatal("Test vector failure")
	}
}

// TestConstructor tests that the constructor can run without error.
func TestConstructor(t *testing.T) {

	if VERBOSE {
		fmt.Println("\n-------------------------------")
		fmt.Println("Starting basic constructor test")
		fmt.Printf("-------------------------------\n\n")
	}

	NewChatter()
}

// DoHandshake executes the three-step handshake process.
// It does not fail on an error, but returns it.
func DoHandshake(t *testing.T, alice, bob *Chatter) error {

	if VERBOSE {
		fmt.Println("Starting handshake sequence")
		fmt.Printf("Initiator identity: %X\n", alice.Identity.Fingerprint()[:HANDLE_LENGTH])
		fmt.Printf("Responder identity: %X\n", bob.Identity.Fingerprint()[:HANDLE_LENGTH])
	}

	aliceShare, err := alice.InitiateHandshake(&bob.Identity.PublicKey)
	if err != nil {
		t.Logf("Error initiating handshake")
		return err
	}

	if VERBOSE {
		fmt.Printf("Initiator sends ephemeral key: %X\n", aliceShare.Fingerprint())
	}

	bobShare, bobCheck, err := bob.ReturnHandshake(&alice.Identity.PublicKey, aliceShare)
	if err != nil {
		t.Logf("Error responding to hanshake")
		return err
	}
	if VERBOSE {
		fmt.Printf("Responder sends ephemeral key: %X\n", bobShare.Fingerprint())
	}

	aliceCheck, err := alice.FinalizeHandshake(&bob.Identity.PublicKey, bobShare)
	if err != nil {
		t.Logf("Error finalizing handshake")
		return err
	}

	if !bytes.Equal(aliceCheck.Key, bobCheck.Key) {
		t.Logf("Handshake participants don't agree on master key")
		return errors.New("Handshake failed")
	}
	if VERBOSE {
		fmt.Printf("Handshake master key hash: %X\n", bobCheck.Key)
	}

	return err
}

// TestHandshake tests if the handshake protocol can run without errors.
func TestHandshake(t *testing.T) {

	if VERBOSE {
		fmt.Println("\n-------------------------------")
		fmt.Println("Starting handshake test")
		fmt.Printf("-------------------------------\n\n")
	}

	FailOnError(t, DoHandshake(t, NewChatter(), NewChatter()))
}

// TestHandshake tests if the handshake protocol can run without errors.
func TestHandshakeVector(t *testing.T) {

	SkipOnError(t, DoHandshake(t, NewChatter(), NewChatter()))

	if VERBOSE {
		fmt.Println("\n-------------------------------")
		fmt.Println("Starting handshake vector test")
		fmt.Printf("-------------------------------\n\n")
	}

	SetFixedRandomness(true)
	defer SetFixedRandomness(false)

	alice := NewChatter()
	bob := NewChatter()

	aliceShare, _ := alice.InitiateHandshake(&bob.Identity.PublicKey)
	_, bobCheck, _ := bob.ReturnHandshake(&alice.Identity.PublicKey, aliceShare)

	CheckTestVector(t, bobCheck.Key, "A72EB9D3D4EF6DAF82B44D1D4F44700226AA37437887922EDD2C55682221D2BA", "Handshake check")
}

// CheckSend creates a message from sender to receiver by calling SendMessage
// on the sender. It does not fail on error, but returns the error.
func CheckSend(t *testing.T,
	sender, receiver *Chatter,
	plaintext string) (*Message, error) {

	if VERBOSE {
		fmt.Printf("0x%0X attempting to send plaintext \"%s\" to 0x%0X\n",
			sender.Identity.Fingerprint()[:HANDLE_LENGTH],
			plaintext,
			receiver.Identity.Fingerprint()[:HANDLE_LENGTH])
	}
	message, err := sender.SendMessage(&receiver.Identity.PublicKey, plaintext)
	if err != nil {
		return nil, err
	}
	if VERBOSE {
		fmt.Printf("Sent with counter: %d, next share: %0X, ciphertext: %0X\n",
			message.Counter,
			message.NextDHRatchet.Fingerprint(),
			message.Ciphertext)
	}

	return message, nil
}

// CheckReceive delivers a message to the receiver. It does not fail on error,
// but returns the error. If intendedPlaintext is specified, it raises an error
// if the message is not decrypted to the specified plaintext.
func CheckReceive(t *testing.T,
	receiver *Chatter,
	message *Message,
	intendedPlaintext string) error {

	if VERBOSE {
		fmt.Printf("0x%0X receiving message from 0x%0X, counter: %d, next share: %0X, ciphertext: %0X\n",
			message.Sender.Fingerprint()[:HANDLE_LENGTH],
			message.Receiver.Fingerprint()[:HANDLE_LENGTH],
			message.Counter,
			message.NextDHRatchet.Fingerprint(),
			message.Ciphertext)
	}
	received, err := receiver.ReceiveMessage(message)
	if err != nil {
		t.Logf("Error receiving message:")
		t.Logf(err.Error())
		return err
	}

	// Check the plaintext is correct, if the target is known.
	if intendedPlaintext != "" && received != intendedPlaintext {
		t.Logf("Message not decrypted successfully")
		return err
	}
	if VERBOSE {
		fmt.Printf("0x%0X decrypted plaintext \"%s\"\n",
			receiver.Identity.Fingerprint()[:HANDLE_LENGTH],
			received)
	}

	return nil
}

// CheckSendReceive generates a message from sender to receiver by
// asking the sender to sent it, then pass the message to the receiver.
// It does not fail on an error, but returns it. It will exit early
// if the Send call returns an error.
func CheckSendReceive(t *testing.T,
	sender, receiver *Chatter,
	plaintext string) error {

	message, err := CheckSend(t, sender, receiver, plaintext)
	if err != nil {
		return err
	}

	return CheckReceive(t, receiver, message, plaintext)
}

// TestOneWayChat tests a simple conversation where only one party (the
// responder) sends a series of messages. If any message raises an error
// or fails to decrypt, the test fails. If the handshake fails the test
// is skipped.
func TestOneWayChat(t *testing.T) {

	alice := NewChatter()
	bob := NewChatter()
	SkipOnError(t, DoHandshake(t, alice, bob))

	if VERBOSE {
		fmt.Println("\n-------------------------------")
		fmt.Println("Starting one-way test sequence")
		fmt.Printf("-------------------------------\n\n")
	}

	for _, m := range []string{"hi Alice!", "you there???", "I miss you ❤️"} {
		FailOnError(t, CheckSendReceive(t, bob, alice, m))
	}
}

// TestOneWayReverseChat tests a conversation where only one party (the
// initiator) sends a series of messages. Note that this is slightly
// harder to implement since a DH ratchet should happen before the message
// is sent. If any message raises an error or fails to decrypt, the test fails.
// If the handshake fails the test is skipped.
func TestOneWayChatReverse(t *testing.T) {

	alice := NewChatter()
	bob := NewChatter()
	SkipOnError(t, DoHandshake(t, alice, bob))

	if VERBOSE {
		fmt.Println("\n-------------------------------")
		fmt.Println("Starting reverse one-way test sequence")
		fmt.Printf("-------------------------------\n\n")
	}

	for _, m := range []string{"Bob!", "sorry my phone died earlier", "I miss you too..."} {
		FailOnError(t, CheckSendReceive(t, alice, bob, m))
	}
}

// TestErrorRecovery tests first that an error is raised if a message is
// modified before delivery. It then tests that the receiver can recover
// from the error and decrypt the correct message if it is sent later.
// If the handshake fails the test is skipped.
func TestErrorRecovery(t *testing.T) {

	alice := NewChatter()
	bob := NewChatter()
	SkipOnError(t, DoHandshake(t, alice, bob))

	if VERBOSE {
		fmt.Println("\n-------------------------------")
		fmt.Println("Starting error recovery test")
		fmt.Printf("-------------------------------\n\n")
	}

	message, err := CheckSend(t, alice, bob, "test")
	SkipOnError(t, err)

	message.Counter += 1
	if _, err = bob.ReceiveMessage(message); err == nil {
		t.Fatal("Did not raise error for modified sequence number")
	}
	message.Counter -= 1

	message.Ciphertext[4] ^= 0x10
	if _, err = bob.ReceiveMessage(message); err == nil {
		t.Fatal("Did not raise error for modified ciphertext")
	}
	message.Ciphertext[4] ^= 0x10

	FailOnError(t, CheckReceive(t, bob, message, "test"))
}

// TestAlternatingChat tests a conversation where both parties
// alternate sending message. If the handshake fails the test is skipped.
func TestAlternatingChat(t *testing.T) {

	alice := NewChatter()
	bob := NewChatter()
	SkipOnError(t, DoHandshake(t, alice, bob))

	if VERBOSE {
		fmt.Println("\n-------------------------------")
		fmt.Println("Starting alternating test sequence")
		fmt.Printf("-------------------------------\n\n")
	}

	FailOnError(t, CheckSendReceive(t, alice, bob, "Roberto!"))
	FailOnError(t, CheckSendReceive(t, bob, alice, "Alicia"))
	FailOnError(t, CheckSendReceive(t, alice, bob, "¿qué pasa?"))
	FailOnError(t, CheckSendReceive(t, bob, alice, "nada"))
	FailOnError(t, CheckSendReceive(t, alice, bob, "😴"))
	FailOnError(t, CheckSendReceive(t, bob, alice, "jajaja"))
}

// TestSynchronousChat tests a conversation where both parties send
// multiple messages in a row. If the handshake fails the test is skipped.
func TestSynchronousChat(t *testing.T) {

	alice := NewChatter()
	bob := NewChatter()
	SkipOnError(t, DoHandshake(t, alice, bob))

	if VERBOSE {
		fmt.Println("\n-------------------------------")
		fmt.Println("Starting short synchronous test sequence")
		fmt.Printf("-------------------------------\n\n")
	}

	FailOnError(t, CheckSendReceive(t, alice, bob, "Hello there!"))
	FailOnError(t, CheckSendReceive(t, bob, alice, "General Kenobi, you are a bold one"))
	FailOnError(t, CheckSendReceive(t, bob, alice, "I find your behavior bewildering..."))
	FailOnError(t, CheckSendReceive(t, bob, alice, "Surely you realize you're doomed"))
	FailOnError(t, CheckSendReceive(t, bob, alice, "Kill him!"))
	FailOnError(t, CheckSendReceive(t, bob, alice, "Back away. I will deal with this Jedi slime myself. "))
	FailOnError(t, CheckSendReceive(t, alice, bob, "Your move"))
	FailOnError(t, CheckSendReceive(t, alice, bob, "..."))
	FailOnError(t, CheckSendReceive(t, bob, alice, " You fool. I have been trained in your Jedi arts by Count Dooku himself."))
	FailOnError(t, CheckSendReceive(t, bob, alice, "Attack, Kenobi."))
	FailOnError(t, CheckSendReceive(t, alice, bob, "You forget I trained the Jedi that defeated Count Dooku!"))
	FailOnError(t, CheckSendReceive(t, alice, bob, "I may not defeat your droids, but my troops certainly will."))
	FailOnError(t, CheckSendReceive(t, bob, alice, "Army or not, you must realize you are doomed."))
	FailOnError(t, CheckSendReceive(t, alice, bob, "I don't think so."))
}

// TestSynchronousChatVector tests a conversation where both parties send
// multiple messages in a row, using fixed keys. It checks if the final
// ciphertext is as expected. If the handshake fails the test is skipped.
func TestSynchronousChatVector(t *testing.T) {

	alice := NewChatter()
	bob := NewChatter()
	SkipOnError(t, DoHandshake(t, alice, bob))

	if VERBOSE {
		fmt.Println("\n-------------------------------")
		fmt.Println("Starting synchronous test vector sequence")
		fmt.Printf("-------------------------------\n\n")
	}

	SetFixedRandomness(true)
	defer SetFixedRandomness(false)
	alice = NewChatter()
	bob = NewChatter()

	FailOnError(t, DoHandshake(t, alice, bob))

	//Check first message
	message, err := CheckSend(t, bob, alice, "Alice?")
	SkipOnError(t, err)

	CheckTestVector(t, message.Sender.Fingerprint(), "83F257B18A903848BA6CDB628E7D925B", "Sender")
	CheckTestVector(t, message.Receiver.Fingerprint(), "7446CB2BE09E4967E72B861EB81BC5AF", "Receiver")
	CheckTestVector(t, message.NextDHRatchet.Fingerprint(), "EF8D206106A74C26DBC3EB4F8679D3DB", "NextDHRatchet")
	CheckTestVector(t, []byte{byte(message.Counter)}, "01", "Counter")
	CheckTestVector(t, []byte{byte(message.LastUpdate)}, "00", "LastUpdate")
	CheckTestVector(t, message.IV, "0102030405060708090A0B0C", "IV")
	CheckTestVector(t, message.Ciphertext, "52D0A0679552808A67C2C5F13A6607CBBFC3FEA30B28", "Ciphertext")

	SkipOnError(t, CheckReceive(t, alice, message, "Alice?"))

	//Check second message
	message, err = CheckSend(t, alice, bob, "Bob...")
	SkipOnError(t, err)

	CheckTestVector(t, message.Sender.Fingerprint(), "7446CB2BE09E4967E72B861EB81BC5AF", "Sender")
	CheckTestVector(t, message.Receiver.Fingerprint(), "83F257B18A903848BA6CDB628E7D925B", "Receiver")
	CheckTestVector(t, message.NextDHRatchet.Fingerprint(), "CE0753ABB34AFC0EDC95B3BF72924E20", "NextDHRatchet")
	CheckTestVector(t, []byte{byte(message.Counter)}, "01", "Counter")
	CheckTestVector(t, []byte{byte(message.LastUpdate)}, "00", "LastUpdate")
	CheckTestVector(t, message.IV, "0102030405060708090A0B0C", "IV")
	CheckTestVector(t, message.Ciphertext, "D338E92B04DAA4F6C25F6AE3952A8EBB46BF29DE9CDB", "Ciphertext")

	SkipOnError(t, CheckReceive(t, bob, message, "Bob..."))

	//Longer sequence, unchecked
	SkipOnError(t, CheckSendReceive(t, bob, alice, "Alice!!"))
	SkipOnError(t, CheckSendReceive(t, bob, alice, "Alice!!!"))
	SkipOnError(t, CheckSendReceive(t, bob, alice, "Alice!!!"))
	SkipOnError(t, CheckSendReceive(t, alice, bob, "Bob!"))
	SkipOnError(t, CheckSendReceive(t, alice, bob, "I heard you the first time"))
	SkipOnError(t, CheckSendReceive(t, alice, bob, "No need to repeat yourself..."))
	SkipOnError(t, CheckSendReceive(t, bob, alice, "Sorry Alice"))
	SkipOnError(t, CheckSendReceive(t, bob, alice, "I got carried away"))
	SkipOnError(t, CheckSendReceive(t, bob, alice, "won't happen again"))
	SkipOnError(t, CheckSendReceive(t, alice, bob, "that's okay Bob"))
	message, err = CheckSend(t, alice, bob, "it happens!")
	SkipOnError(t, err)

	// Check final message after extended conversation
	CheckTestVector(t, message.Sender.Fingerprint(), "7446CB2BE09E4967E72B861EB81BC5AF", "Sender")
	CheckTestVector(t, message.Receiver.Fingerprint(), "83F257B18A903848BA6CDB628E7D925B", "Receiver")
	CheckTestVector(t, message.NextDHRatchet.Fingerprint(), "34FAB4CF6AE3CFB23A9AF2C0ECE3C4E2", "NextDHRatchet")
	CheckTestVector(t, []byte{byte(message.Counter)}, "06", "Counter")
	CheckTestVector(t, []byte{byte(message.LastUpdate)}, "04", "LastUpdate")
	CheckTestVector(t, message.IV, "0102030405060708090A0B0C", "IV")
	CheckTestVector(t, message.Ciphertext, "8E3E9C653B7DF0CA5613F4DB3ADC895FEA6CEDFDA4C7E3CD31070A", "Ciphertext")
}

// TestTeardown tests that a session can be ended by calling
// EndSession, after which no messages should be sent.
// It then tests that a new handshake can be completed and messages sent again.
// If the first handshake fails the test is skipped.
func TestTeardown(t *testing.T) {
	alice := NewChatter()
	bob := NewChatter()

	SkipOnError(t, DoHandshake(t, alice, bob))
	SkipOnError(t, CheckSendReceive(t, alice, bob, "Ping"))
	SkipOnError(t, CheckSendReceive(t, bob, alice, "Pong"))

	FailOnError(t, alice.EndSession(&bob.Identity.PublicKey))
	if _, err := CheckSend(t, alice, bob, "Ping?"); err == nil {
		t.Fatal("Should not be able to send messages after ending sesion.")
	}
	if _, err := CheckSend(t, bob, alice, "Ping?"); err != nil {
		t.Fatal("Should be able to send messages to partner who has closed session.")
	}
	FailOnError(t, bob.EndSession(&alice.Identity.PublicKey))

	if alice.EndSession(&bob.Identity.PublicKey) == nil {
		t.Fatal("Session should not be ended twice")
	}

	FailOnError(t, DoHandshake(t, alice, bob))
	FailOnError(t, CheckSendReceive(t, bob, alice, "Pong"))
	FailOnError(t, CheckSendReceive(t, alice, bob, "Ping"))
}

// SetupChatters creates an array of n chatters, initializing them
// and performing a handshake between all pairs. Note that this requires
// n^2 handshakes.
func SetupChatters(t *testing.T, n int) ([]*Chatter, error) {

	chatters := make([]*Chatter, n)
	for i := 0; i < len(chatters); i++ {
		chatters[i] = NewChatter()
		if VERBOSE {
			fmt.Printf("Created new chatter #%d: %0X\n",
				i,
				chatters[i].Identity.Fingerprint()[:HANDLE_LENGTH])
		}
	}

	for i := 0; i < len(chatters); i++ {
		for j := i + 1; j < len(chatters); j++ {
			var err error
			if (i^j)&1 == 1 {
				err = DoHandshake(t, chatters[i], chatters[j])
			} else {
				err = DoHandshake(t, chatters[j], chatters[i])
			}
			if err != nil {
				return chatters, err
			}
		}
	}

	return chatters, nil
}

// TestSynchronousChatExtended creates an array of EXTENDED_TEST_PARTICIPANTS
// chatters and then sends a message between a random pair for
// EXTENDED_TEST_ROUNDS. If the setup fails, the test is skipped.
func TestSynchronousChatExtended(t *testing.T) {

	if testing.Short() {
		t.Skip("Skipping extended text in short mode.")
	}

	chatters, err := SetupChatters(t, EXTENDED_TEST_PARTICIPANTS)
	SkipOnError(t, err)

	if VERBOSE {
		fmt.Println("\n-------------------------------")
		fmt.Printf("Starting extended synchronous testing, %d participants, %d rounds\n",
			EXTENDED_TEST_PARTICIPANTS,
			EXTENDED_TEST_ROUNDS)
		fmt.Printf("-------------------------------\n\n")
	}

	for i := 0; i < EXTENDED_TEST_ROUNDS; i++ {

		c1 := chatters[rand.Int()%len(chatters)]
		c2 := chatters[rand.Int()%len(chatters)]
		if c1 == c2 {
			continue
		}
		m := fmt.Sprintf("M%d", i)
		if VERBOSE {
			fmt.Printf("Message \"%s\" to be delivered from 0x%0X to 0x%0X\n",
				m,
				c1.Identity.Fingerprint()[:HANDLE_LENGTH],
				c2.Identity.Fingerprint()[:HANDLE_LENGTH])
		}

		FailOnError(t, CheckSendReceive(t, c1, c2, m))
	}
}

// SendQueuedMessage generates a new message by calling SendMessage and
// adds it to the queue q in position i.
func SendQueuedMessage(t *testing.T,
	q []*Message,
	i int,
	sender, receiver *Chatter,
	plaintext string) error {

	message, err := CheckSend(t, sender, receiver, plaintext)
	if err != nil {
		return err
	}

	if VERBOSE {
		fmt.Printf("Message \"%s\" from 0x%0X to 0x%0X sent and added to queue\n",
			plaintext,
			sender.Identity.Fingerprint()[:HANDLE_LENGTH],
			receiver.Identity.Fingerprint()[:HANDLE_LENGTH])
	}

	q[i] = message
	return nil
}

// DeliverQueuedMessage takes the specified message from the queue and
// delivers it to the intended recipient. If deliveryError is set,
// the message is delivered with error and not removed from the queue.
func DeliverQueuedMessage(t *testing.T,
	c map[PublicKey]*Chatter,
	q []*Message,
	i int,
	deliveryError bool) error {
	if VERBOSE {
		fmt.Printf("Message %d from 0x%0X delivered to 0x%0X\n",
			q[i].Counter,
			q[i].Sender.Fingerprint()[:HANDLE_LENGTH],
			q[i].Receiver.Fingerprint()[:HANDLE_LENGTH])
		if deliveryError {
			fmt.Println("*******Delivery error induced*********")
		}
	}

	if deliveryError {
		q[i].Ciphertext[3] ^= 0x08
		if _, err := c[*q[i].Receiver].ReceiveMessage(q[i]); err == nil {
			t.Fatal("Did not raise error for modified ciphertext")
		}
		q[i].Ciphertext[3] ^= 0x08
		return nil
	}

	return CheckReceive(t, c[*q[i].Receiver], q[i], "")
}

// TestAsynchronousChat tests a short chat sequence between Alice and Bob with
// many message delayed and delivered out of order. No delivery errors occur.
// If the setup fails, the test is skipped.
func TestAsynchronousChat(t *testing.T) {

	alice := NewChatter()
	bob := NewChatter()
	SkipOnError(t, DoHandshake(t, alice, bob))

	if VERBOSE {
		fmt.Println("\n-------------------------------")
		fmt.Println("Starting short asynchronous test")
		fmt.Printf("-------------------------------\n\n")
	}

	queue := make([]*Message, 9)

	c := make(map[PublicKey]*Chatter)
	c[alice.Identity.PublicKey] = alice
	c[bob.Identity.PublicKey] = bob

	SendQueuedMessage(t, queue, 1, alice, bob, "1")
	SendQueuedMessage(t, queue, 2, alice, bob, "2")
	SendQueuedMessage(t, queue, 3, bob, alice, "3")
	SendQueuedMessage(t, queue, 4, alice, bob, "4")
	SendQueuedMessage(t, queue, 5, bob, alice, "5")
	FailOnError(t, DeliverQueuedMessage(t, c, queue, 5, false))
	FailOnError(t, DeliverQueuedMessage(t, c, queue, 3, false))
	FailOnError(t, DeliverQueuedMessage(t, c, queue, 4, false))
	SendQueuedMessage(t, queue, 6, bob, alice, "6")
	SendQueuedMessage(t, queue, 7, bob, alice, "7")
	SendQueuedMessage(t, queue, 8, alice, bob, "8")

	FailOnError(t, DeliverQueuedMessage(t, c, queue, 7, false))
	FailOnError(t, DeliverQueuedMessage(t, c, queue, 6, false))
	FailOnError(t, DeliverQueuedMessage(t, c, queue, 8, false))
	FailOnError(t, DeliverQueuedMessage(t, c, queue, 2, false))
	FailOnError(t, DeliverQueuedMessage(t, c, queue, 1, false))

	if err := DeliverQueuedMessage(t, c, queue, 1, false); err == nil {
		t.Fatal("Accepted replay of late message without error")
	}
}

// TestAsynchronousChatExtended creates an array of EXTENDED_TEST_PARTICIPANTS
// chatters. In each round, it then randomly either enqueues a sent message
// from a random chatter to another, or picks a message from the queue
// and delivers it. This runs for EXTENDED_TEST_ROUNDS. Errors are induced with
// probability EXTENDED_TEST_ERROR_RATE, set this above zero to test error
// recovery. If the setup fails, the test is skipped.
func TestAsynchronousChatExtended(t *testing.T) {

	if testing.Short() {
		t.Skip("Skipping extended text in short mode.")
	}

	chatters, err := SetupChatters(t, EXTENDED_TEST_PARTICIPANTS)
	SkipOnError(t, err)

	if VERBOSE {
		fmt.Println("\n-------------------------------")
		fmt.Printf("Starting extended asynchronous testing, %d participants, %d rounds\n",
			EXTENDED_TEST_PARTICIPANTS,
			EXTENDED_TEST_ROUNDS)
		fmt.Printf("-------------------------------\n\n")
	}

	c := make(map[PublicKey]*Chatter)
	for i := range chatters {
		c[chatters[i].Identity.PublicKey] = chatters[i]
	}

	queue := make([]*Message, 100)
	queueLength := 0

	for i := 0; i < EXTENDED_TEST_ROUNDS; i++ {
		if queueLength < EXTENDED_TEST_ROUNDS-i && queueLength < len(queue) && rand.Int()%2 == 0 {

			c1 := chatters[rand.Int()%len(chatters)]
			c2 := chatters[rand.Int()%len(chatters)]
			if c1 == c2 {
				continue
			}
			SendQueuedMessage(t, queue, queueLength, c1, c2, fmt.Sprintf("M %d", i))
			queueLength += 1
		} else if queueLength > 0 {

			// deliver a random queued message
			j := rand.Int() % queueLength
			deliveryError := (rand.Float64() < EXTENDED_TEST_ERROR_RATE)
			FailOnError(t, DeliverQueuedMessage(t, c, queue, j, deliveryError))
			if !deliveryError {
				if queueLength > 1 && j < queueLength-1 {
					queue[j] = queue[queueLength-1]
				}
				queueLength -= 1
			}
		}
	}
}
