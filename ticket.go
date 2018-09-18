
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"bytes"
	// "reflect"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
)

var logger = shim.NewLogger("ExchainChaincode")

const(
	MD_office = iota
	HANA
	SMB
	IBS
	S4_HANA
	GS
	SF
	IoT

	//numberOfLoBs
)

const(
	P0 = iota
	P1
)

const(
	Created = iota
	Applied
	Ongoing
	Done
)


type Participant struct {
	UserID		string 		`json:"Participant_UserID"`
	UserName    string 		`json:"Participant_UserName"`
	Password    string  	`json:"Participant_Password"`
	
	IsAdmin     bool 		`json:"Participant_IsAdmin"`
	LoB			int     	`json:"Participant_LoB"`
}

type Credit struct {
	UserID		string  	`json:"Credit_UserID"`
	Value       int     	`json:"Credit_Value"`
	
	TicketIDs   []string 	`json:"Credit_TicketIDs"`
}

type Ticket struct {
	TicketID	string 		`json:"Ticket_TicketID"`
	Status		int 		`json:"Ticket_Status"`

	Title		string 		`json:"Ticket_Title"`
	Type        int 		`json:"Ticket_Type"`

	Value		int 		`json:"Ticket_Value"`
	UserID		string		`json:"Ticket_UserID"`

	DeadLine	time.Time 	`json:"Ticket_Deadline"`
	Comment		string     	`json:"Ticket_Comment"`
	Policy		string 		`json:"Ticket_Policy"`
}

type Order struct {
	TicketID 	string		`json:"TicketID"`
	UserID		string		`json:"UserID"`
	Status		int			`json:"Status"`
}

//SmartContract - Chaincode for asset Reading
type SmartContract struct {
}

//ReadingIDIndex - Index on IDs for retrieval all Readings
type ReadingIDIndex struct {
	UserIDs []string 		`json:"UserIDs"`
}

func main() {
	err := shim.Start(new(SmartContract))
	if err != nil {
		fmt.Printf("Error starting Exchain chaincode function main(): %s", err)
	} else {
		fmt.Printf("Starting Exchain chaincode function main() executed successfully")
	}
}

//Init - The chaincode Init function: No arguments, only initializes a ID array as Index for retrieval of all Readings
func (rdg *SmartContract) Init(stub shim.ChaincodeStubInterface) peer.Response {
	var readingIDIndex ReadingIDIndex
	bytes, _ := json.Marshal(readingIDIndex)
	stub.PutState("readingIDIndex", bytes)
	return shim.Success(nil)
}

//Invoke - The chaincode Invoke function:
func (rdg *SmartContract) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	function, args := stub.GetFunctionAndParameters()
	logger.Info(" ****** Invoke: function: ", function)

	switch function{
	case "addParticipant":
		return rdg.addParticipant(stub, args)
	case "readParticipant":
		return rdg.readParticipant(stub, args[0])
	case "updateParticipant":
		return rdg.updateParticipant(stub, args)
	case "deleteParticipant":
		return rdg.deleteParticipant(stub, args[0])
	
	case "CreditCreate":
		return rdg.CreditCreate(stub, args)
	case "CreditRead":
		return rdg.CreditRead(stub, args[0])
	case "CreditUpdate":
		return rdg.CreditUpdate(stub, args)
	case "CreditDelete":
		return rdg.CreditDelete(stub, args[0])

	case "TicketCreate":
		return rdg.TicketCreate(stub, args)
	case "TicketRead":
		return rdg.TicketRead(stub, args)
	case "TicketUpdate":
		return rdg.TicketUpdate(stub, args)
	case "TicketDelete":
		return rdg.TicketDelete(stub, args[0])
	
	case "OrderApply":
		return rdg.OrderCreate(stub, args)
	case "OrderRead":
		return rdg.OrderRead(stub, args)
	case "OrderRead2":
		return rdg.OrderRead2(stub, args)
	case "OrderUpdate":
		return rdg.OrderUpdate(stub, args)


	case "history":
		return rdg.TestGetHistoryTicket(stub, args)
	default:
		logger.Error("Received unknown function invocation: ", function)
	}
	return shim.Error("Received unknown function invocation")
}

//getReadingFromArgs - construct a reading structure from string array of arguments
func getParticipantFromArgs(args []string) (participant Participant, err error) {

	if  strings.Contains(args[0], "\"Participant_UserName\"") == false ||
		strings.Contains(args[0], "\"Participant_UserID\"")   == false ||
		strings.Contains(args[0], "\"Participant_Password\"") == false ||
		strings.Contains(args[0], "\"Participant_IsAdmin\"")  == false ||
		strings.Contains(args[0], "\"Participant_LoB\"")      == false   {
		return participant, errors.New("Unknown field: Input JSON does not comply to schema")
	}

	err = json.Unmarshal([]byte(args[0]), &participant)
	if err != nil {
		return participant, err
	}
	return participant, nil
}

//Invoke Route: addNewReading
func (rdg *SmartContract) addParticipant(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	participant, err := getParticipantFromArgs(args)
	if err != nil {
		return shim.Error("Reading participant is Corrupted")
	}
	record, err := stub.GetState(participant.UserID)
	if record != nil {
		return shim.Error("This participant already exists: " + participant.UserID)
	}
	_, err = rdg.saveParticipant(stub, participant)
	if err != nil {
		return shim.Error(err.Error())
	}
	_, err = rdg.updateReadingIDIndex(stub, participant)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}

//Helper: Save purchaser
func (rdg *SmartContract) saveParticipant(stub shim.ChaincodeStubInterface, participant Participant) (bool, error) {
	bytes, err := json.Marshal(participant)
	if err != nil {
		return false, errors.New("Error converting reading record JSON")
	}
	err = stub.PutState(participant.UserID, bytes)
	if err != nil {
		return false, errors.New("Error storing Reading record")
	}
	return true, nil
}

//Helper: Update reading Holder - updates Index
func (rdg *SmartContract) updateReadingIDIndex(stub shim.ChaincodeStubInterface, participant Participant) (bool, error) {
	var participantIDs ReadingIDIndex
	bytes, err := stub.GetState("readingIDIndex")
	if err != nil {
		return false, errors.New("updateReadingIDIndex: Error getting readingIDIndex array Index from state")
	}
	err = json.Unmarshal(bytes, &participantIDs)
	if err != nil {
		return false, errors.New("updateReadingIDIndex: Error unmarshalling readingIDIndex array JSON")
	}
	participantIDs.UserIDs = append(participantIDs.UserIDs, participant.UserID)
	bytes, err = json.Marshal(participantIDs)
	if err != nil {
		return false, errors.New("updateReadingIDIndex: Error marshalling new participant ID")
	}
	err = stub.PutState("readingIDIndex", bytes)
	if err != nil {
		return false, errors.New("updateReadingIDIndex: Error storing new participant ID in readingIDIndex (Index)")
	}
	return true, nil
}


//Query Route: readReading
func (rdg *SmartContract) readParticipant(stub shim.ChaincodeStubInterface, participantID string) peer.Response {
	participantAsByteArray, err := rdg.retrieveParticipant(stub, participantID)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(participantAsByteArray)
}

//Helper: Retrieve purchaser
func (rdg *SmartContract) retrieveParticipant(stub shim.ChaincodeStubInterface, participantID string) ([]byte, error) {
	var participant Participant
	var participantAsByteArray []byte
	bytes, err := stub.GetState(participantID)

	if err != nil {
		return participantAsByteArray, errors.New("retrieveParticipant: Error retrieving participant with ID: " + participantID)
	}
	err = json.Unmarshal(bytes, &participant)
	if err != nil {
		return participantAsByteArray, errors.New("retrieveParticipant: Corrupt reading record " + string(bytes))
	}
	participantAsByteArray, err = json.Marshal(participant)
	if err != nil {
		return participantAsByteArray, errors.New("readParticipant: Invalid participant Object - Not a valid JSON")
	}
	return participantAsByteArray, nil
}


//Helper: Reading readingStruct //change template
func (rdg *SmartContract) deleteParticipant(stub shim.ChaincodeStubInterface, participantID string) peer.Response {
	_, err := rdg.retrieveParticipant(stub, participantID)
	if err != nil {
		return shim.Error(err.Error())
	}
	err = stub.DelState(participantID)
	if err != nil {
		return shim.Error(err.Error())
	}
	_, err = rdg.deleteReadingIDIndex(stub, participantID)
		if err != nil {
			return shim.Error(err.Error())
		}
	return shim.Success(nil)
}

//Helper: delete ID from readingStruct Holder
func (rdg *SmartContract) deleteReadingIDIndex(stub shim.ChaincodeStubInterface, participantID string) (bool, error) {
	var participantIDs ReadingIDIndex
	bytes, err := stub.GetState("readingIDIndex")
	if err != nil {
		return false, errors.New("deleteReadingIDIndex: Error getting readingIDIndex array Index from state")
	}
	err = json.Unmarshal(bytes, &participantIDs)
	if err != nil {
		return false, errors.New("deleteReadingIDIndex: Error unmarshalling readingIDIndex array JSON")
	}
	participantIDs.UserIDs, err = deleteKeyFromStringArray(participantIDs.UserIDs, participantID)
	if err != nil {
		return false, errors.New(err.Error())
	}
	bytes, err = json.Marshal(participantIDs)
	if err != nil {
		return false, errors.New("deleteReadingIDIndex: Error marshalling new readingStruct ID")
	}
	err = stub.PutState("readingIDIndex", bytes)
	if err != nil {
		return false, errors.New("deleteReadingIDIndex: Error storing new readingStruct ID in readingIDIndex (Index)")
	}
	return true, nil
}

//deleteKeyFromArray
func deleteKeyFromStringArray(array []string, key string) (newArray []string, err error) {
	for _, entry := range array {
		if entry != key {
			newArray = append(newArray, entry)
		}
	}
	if len(newArray) == len(array) {
		return newArray, errors.New("Specified Key: " + key + " not found in Array")
	}
	return newArray, nil
}

//Invoke Route: updateParticipant
func (rdg *SmartContract) updateParticipant(stub shim.ChaincodeStubInterface, args []string) peer.Response {

	var currParticipant Participant
	newParticipant, err := getParticipantFromArgs(args)
	participantAsByteArray, err := rdg.retrieveParticipant(stub, newParticipant.UserID)
	if err != nil {
		return shim.Error(err.Error())
	}

	err = json.Unmarshal(participantAsByteArray, &currParticipant)
	if err != nil {
		return shim.Error("updateReading: Error unmarshalling readingStruct array JSON")
	}

	_, err = rdg.saveParticipant(stub, newParticipant)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}

func (rdg *SmartContract) CreditCreate(stub shim.ChaincodeStubInterface, args []string) peer.Response {

	// ==== Check whether the participant already exsites. ====
	record, err := stub.GetState("Credit_UerID_"+args[0])

	if record != nil {
		return shim.Error("This participant's:%s credit has already existed." + args[0])
	}

	userID := args[0]
	value, err := strconv.Atoi(args[1])

	err = CreditInit(stub, userID, value)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func CreditInit(stub shim.ChaincodeStubInterface, userID string, value int) error{
	var credit Credit

	// ==== Create Credit object and Credit to JSON ====
	credit = Credit{UserID: userID, Value: value}

	creditAsByteArray, err := json.Marshal(credit)
	if err != nil {
		return errors.New(err.Error())
	}

	// ==== Save Credit to state ====
	err = stub.PutState("Credit_UerID_"+userID, creditAsByteArray)
	if err != nil {
		return errors.New(err.Error())
	}

	return nil
}

func (rdg *SmartContract) CreditRead(stub shim.ChaincodeStubInterface, UserID string) peer.Response {
	//to do
	creditAsByteArray, err := retrieveSingleCreditAsByteArray(stub, "Credit_UerID_"+UserID)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(creditAsByteArray)
}

func retrieveSingleCredit(stub shim.ChaincodeStubInterface, creditID string) (Credit, error){
	var credit Credit
	var creditAsByteArray []byte
	var err error

	creditAsByteArray, err = stub.GetState(creditID)

	if err != nil {
		return credit, errors.New("CreditRead: Error credit read participant with ID: " + creditID)
	}
	// else if creditAsBytes == nil {
    //  logger.Error("CreditRead:  Corrupt reading record ", err.Error())
    //  return nil, errors.New("CreditRead: Credit does not exist " + creditID)
    // }

	// For log printing credit Information & check whether the credit does exist
	err = json.Unmarshal(creditAsByteArray, &credit)
	if err != nil {
		return credit, errors.New("CreditRead: Credit does not exist "  + string(creditAsByteArray))
	}
	// For log printing credit Information

	return credit, nil
}

func retrieveSingleCreditAsByteArray(stub shim.ChaincodeStubInterface, creditID string) ([]byte, error){
	var credit Credit
	var creditAsByteArray []byte
	var err error

	logger.Info("-----retrieveSingleCreditAsByteArray :creditID---------", creditID)
	creditAsByteArray, err = stub.GetState(creditID)

	if err != nil {
		return nil, errors.New("CreditRead: Error credit read participant with ID: " + creditID)
	}
	// else if creditAsBytes == nil {
    //  logger.Error("CreditRead:  Corrupt reading record ", err.Error())
    //  return nil, errors.New("CreditRead: Credit does not exist " + creditID)
    // }

	// For log printing credit Information & check whether the credit does exist
	err = json.Unmarshal(creditAsByteArray, &credit)
	if err != nil {
		return nil, errors.New("CreditRead: Credit does not exist "  + string(creditAsByteArray))
	}
	// For log printing credit Information

	logger.Info("-----retrieveSingleCreditAsByteArray---------", credit)

	return creditAsByteArray, nil
}


func (rdg *SmartContract) CreditUpdate(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	var credit Credit
	// ==== Check whether the number of args is 3 ====
	if len(args) != 3 {
		return shim.Error("CreditUpdate: Incorrect number of arguments, Expecting 3")
	}

	// ==== Assign value to variable ====
	userID := args[0]
	value, err := strconv.Atoi(args[1])
	if err != nil {
		return shim.Error("CreditUpdate: Incorrect value :" + args[1])
	}
	ticketID := args[2]
	
	// === Check whether the credit already exist. ====
	creditAsByteArray, err := stub.GetState("Credit_UerID_"+userID)
	if err != nil {
		return shim.Error("CreditUpdate: Failed to get credit :" + err.Error())
	} else if creditAsByteArray == nil {
		errs := fmt.Sprintf("CreditUpdate: Credit_UerID_%s does not exist.", userID)
		logger.Info(" ****** " + errs)
		return shim.Error(errs)
	}

	err = json.Unmarshal(creditAsByteArray, &credit)
	if err != nil {
		return shim.Error(err.Error())
	}

	// === check whether the ticket has been add ===
	if ok := Is_Inarray(credit.TicketIDs, ticketID); ok {
		return shim.Error("CreditUpdate: This ticket has been existed.")
	}

	credit.Value += value
	credit.TicketIDs = append(credit.TicketIDs, ticketID)

	creditAsByteArray, err = json.Marshal(credit)
	if err != nil {
		return shim.Error("CreditUpdate: " + err.Error())
	}

	err = stub.PutState("Credit_UerID_"+credit.UserID, creditAsByteArray)
	return shim.Success(creditAsByteArray)
}

func Is_Inarray(target []string, now string) bool {
	for _, entry := range target {
		if entry == now {
			return true
		}
	}
	return false
}


func (rdg *SmartContract) CreditDelete(stub shim.ChaincodeStubInterface, userID string) peer.Response {
	logger.Info(" ****** CreditDelete start ****** userID:" + userID)
	err := stub.DelState("Credit_UerID_"+userID)
	if err!= nil {
		return shim.Error("CreditDelete: Failed to delete Credit state: " + err.Error())
	}
	
	//Log process for debug
	credit, err := stub.GetState("Credit_UerID_"+userID)
	logger.Info(" ****** CreditDelete ****** " + string(credit))

	return shim.Success(nil)
}

func getTicketFromArgs(args string)(ticket Ticket, err error) {
	if strings.Contains(args, "\"Ticket_TicketID\"") == false	|| 
	strings.Contains(args, "\"Ticket_Title\"") == false 			|| 
	strings.Contains(args, "\"Ticket_Value\"") == false 			||
	strings.Contains(args, "\"Ticket_UserID\"") == false 		||
	strings.Contains(args, "\"Ticket_Title\"") == false {
		return ticket, errors.New("Unknown field: Input JSON does not comly to schema")
	}
	
	err = json.Unmarshal([]byte(args), &ticket)
	if err != nil {
		return ticket, err
	}

	return ticket, nil
}

func saveTicket(stub shim.ChaincodeStubInterface, ticket Ticket)(bool, error) {
	ticketAsBytes, err := json.Marshal(ticket)
	if err != nil {
		return false, errors.New("saveTicket: " + err.Error())
	}
	err = stub.PutState(ticket.TicketID, ticketAsBytes)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (sc *SmartContract)TicketCreate(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// ==== Get ticket from args ====
	// todo 
	// ticket := Ticket{
	// 	TicketID: "ticket_1", 
	// 	Status: 0, 
	// 	Title: "The first Ticket",
	// 	Value: 30,
	// 	UserID: "1",
	// 	DeadLine: time.Now()}
	
	ticket, err := getTicketFromArgs(args[0])
	if err != nil {
		return shim.Error("TicketCreate: " + err.Error())
	}

	// ==== Judge if the ticket already exists ====
	record, err := stub.GetState(ticket.TicketID)
	if record != nil {
		return shim.Error("TicketCreate: The ticket already exists.")
	}
	
	// ==== Put the ticket into ledger ====
	_, err = saveTicket(stub, ticket)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}


func (sc *SmartContract)TicketDelete(stub shim.ChaincodeStubInterface, ticketID string) peer.Response {
	// ==== Judge if the ticket already exists ====
	var ticket Ticket
	ticketAsBytes, err := stub.GetState(ticketID)
	if err != nil {
		return shim.Error(err.Error())
	}

	err = json.Unmarshal(ticketAsBytes, &ticket)
	if err != nil {
		return shim.Error(err.Error())
	}
	logger.Info(" ****** TicketDelete:", ticket)

	err = stub.DelState(ticketID)
	return shim.Success(nil)
}


func (sc *SmartContract)TicketUpdate(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// ==== Get ticket from args ====
	// todo 
	// ticket := Ticket{
	// 	TicketID: "ticket_1", 
	// 	Status: 1, 
	// 	Title: "The first Ticket 1111",
	// 	Value: 20,
	// 	UserID: "111",
	// 	DeadLine: time.Now()}
	
	if len(args) != 2 {
		return shim.Error("TicketUpdate: The number of arguments is not equal to 2.")
	}

	participantID := args[0]
	ticket, err := getTicketFromArgs(args[1])
	if err != nil {
		return shim.Error("TicketUpdate: " + err.Error())
	}
	// ==== Judge if the ticket already exists ====
	record, err := stub.GetState(ticket.TicketID)
	if record == nil {
		return shim.Error("TicketCreate: The ticket does not exist.")
	}

	if participantID != ticket.UserID {
		return shim.Error("TicketUpdate: You have no rights to update the ticket")
	}

	// ==== Update the ledger ====
	_, err = saveTicket(stub, ticket)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}


func (sc *SmartContract)TicketRead(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// ==== Read ticket from ledger ==== 
	var ticket Ticket
	ticketAsBytes, err := stub.GetState(args[0])
	if err != nil {
		return shim.Error(err.Error())
	}

	err = json.Unmarshal(ticketAsBytes, &ticket)
	if err != nil {
		return shim.Error(err.Error())
	}
	logger.Info(" ****** TicketDelete:", ticket)
	return shim.Success(ticketAsBytes)
}

func (sc *SmartContract) TestGetHistoryTicket(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	ticketInterator, err := stub.GetHistoryForKey("xxx1")
	if err != nil {
		return shim.Error(err.Error())
	}

	// defer ticketInterator.Close()

	for ticketInterator.HasNext() {
		queryResponse, err := ticketInterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		logger.Info("------Test----" + queryResponse.String())
		item, _ := json.Marshal(queryResponse)
		logger.Info("------Test 1----" + string(item))

	}


	return shim.Success(nil)
}

func (sc *SmartContract) OrderCreate(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	ticketID := args[0]
	userID := args[1]

	key, _ := stub.CreateCompositeKey("Order", []string{ticketID, userID})
	logger.Info("------OrderCreate:" + key)

	order := Order{
		TicketID: ticketID,
		UserID: userID,
		Status: 0}
	orderAsByte, _ := json.Marshal(order)
	stub.PutState(key, orderAsByte)

	return shim.Success(nil)
}

func (sc *SmartContract) OrderRead(stub shim.ChaincodeStubInterface, args []string) peer.Response{
	ticketID := args[0]
	userID := args[1]

	key, _ := stub.CreateCompositeKey("Order", []string{ticketID, userID})
	orderAsByte, _ := stub.GetState(key)

	var order Order
	_ = json.Unmarshal(orderAsByte, &order)

	return shim.Success(orderAsByte)
}

func (sc *SmartContract) OrderRead2(stub shim.ChaincodeStubInterface, args []string) peer.Response{
	ticketID := args[0]

	orderInterator, _ := 
	stub.GetStateByPartialCompositeKey("Order", []string{ticketID})

	var buffer bytes.Buffer
	buffer.WriteString("[")
	bArrayMemberAlreadyWritten := false

	for orderInterator.HasNext() {
		queryResponse, err := orderInterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		logger.Info("------Test----" + queryResponse.String())
		item, _ := json.Marshal(queryResponse)
		logger.Info("------Test x----" + string(item))

		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}

		buffer.WriteString("{")
		buffer.WriteString("\"Record\":")
		// Record is a JSON object, so we write as-is
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")

	}
	buffer.WriteString("]")
	
	return shim.Success(buffer.Bytes())
}

func OrderSaving(stub shim.ChaincodeStubInterface, order Order) ([]byte, error) {
	bytes, err := json.Marshal(order)
	if err != nil {
		return nil, err
	}
	key, _ := stub.CreateCompositeKey(
		"Order",
		[]string{order.TicketID, order.UserID})

	err = stub.PutState(key, bytes)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func OrderBlukUpdate(stub shim.ChaincodeStubInterface, ticketID interface{}, userID_array []interface{}, status int) (bool, error) {
	for _, userID := range userID_array {
		order := Order{
			TicketID: ticketID.(string),
			UserID: userID.(string),
			Status: status}
		if status != 0{
			key, _ := stub.CreateCompositeKey("Order", []string{ticketID.(string), userID.(string)})
			orderAsByte, _ := stub.GetState(key)

			var order Order
			_ = json.Unmarshal(orderAsByte, &order)

			if order.Status != status - 1{
				OrderSaving(stub, order)
			}
		} else {
			OrderSaving(stub, order)
		}
	}
	return true, nil
}

func award(stub shim.ChaincodeStubInterface, ticketID string, userID_array []interface{}, value int)(bool, error){
	for _, userID := range userID_array {
		logger.Info("-----xxx---------", "Credit_UerID_"+userID.(string))
		credit, _ := retrieveSingleCredit(stub, "Credit_UerID_"+userID.(string))
		credit.Value += value
		credit.TicketIDs = append(credit.TicketIDs, ticketID)
		logger.Info("-----xxx---------", credit)
		creditAsByteArray, _ := json.Marshal(credit)
		stub.PutState("Credit_UerID_"+credit.UserID, creditAsByteArray)
	}
	return true, nil
}

func (sc *SmartContract) OrderUpdate(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	var raw map[string]interface{}

	err := json.Unmarshal([]byte(args[0]), &raw)
	if err != nil {
		return shim.Error(err.Error())
	}

	var data []interface{}
	var status = 0
	ticketID := raw["TicketID"]
	confirm := raw["Confirm"]
	close := raw["Close"]
	done := raw["Done"]
	awarded := raw["Award"]

	if ticketID == nil {
		return shim.Error("OrderUpdate: TicketID is needed")
	}
	if close != nil {
		_, err = OrderBlukUpdate(stub, ticketID, close.([]interface{}), status)
		if err != nil {
			return shim.Error("OrderUpdate:" + err.Error())
		}
	}

	if confirm != nil {
		data = confirm.([]interface{})
		status = 2
		_, err = OrderBlukUpdate(stub, ticketID, data, status)
	} else if done != nil {
		data = done.([]interface{})
		status = 3
		_, err = OrderBlukUpdate(stub, ticketID, data, status)
	} else if awarded != nil {
		data = awarded.([]interface{})
		status = 4
		_, err = OrderBlukUpdate(stub, ticketID, data, status)
		
		logger.Info("-----1---------")
		// Award user
		var ticket Ticket
		ticketAsBytes, err := stub.GetState(ticketID.(string))
		if err != nil {
			return shim.Error(err.Error())
		}

		err = json.Unmarshal(ticketAsBytes, &ticket)
		if err != nil {
			return shim.Error(err.Error())
		}
		award(stub, ticket.TicketID, data, ticket.Value)
	}

	if err != nil {
		return shim.Error("OrderUpdate:" + err.Error())
	}

	// check ticket status
	// to do 

	return shim.Success(nil)
}

