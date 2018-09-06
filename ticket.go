package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

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
	Owner		Participant `json:"Ticket_Owner"`
	DeadLine	time.Time 	`json:"Ticket_Deadline"`
	Comment		string     	`json:"Ticket_Comment"`
	Policy		string 		`json:"Ticket_Policy"`
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
	logger.Info("function: ", function)
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
	logger.Info("-----retrieveParticipant 1----", bytes)
	logger.Info("-----retrieveParticipant 2----", err)
	if err != nil {
		return participantAsByteArray, errors.New("retrieveParticipant: Error retrieving participant with ID: " + participantID)
	}
	err = json.Unmarshal(bytes, &participant)
	logger.Info("-----retrieveParticipant 3----", participant)
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

func  (rdg *SmartContract) CreditCreate(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	//to do
	//Check whether the participant exsited before.
	record, err := stub.GetState(args[0])

	if record != nil {
		return shim.Error("This participant's:%s credit has already existed." + args[0])
	}

	value, err := strconv.Atoi(args[1])
	credit := Credit{UserID:args[0], Value:value}

	bytes, err := json.Marshal(credit)
	if err != nil {
		return shim.Error(errors.New("CreditCreate: Error marshalling new credit").Error())
	}

	err = stub.PutState(credit.UserID, bytes)

	if err != nil {
		return shim.Error(errors.New("CreditCreate").Error())
	}

	return shim.Success(nil)
}


func  (rdg *SmartContract) CreditRead(stub shim.ChaincodeStubInterface, participantID string) peer.Response {
	creditAsBytes, err := stub.GetState(participantID)
	if err != nil {
		logger.Error("CreditRead:  Corrupt reading record ", err.Error())
		return shim.Error(errors.New("CreditRead: Fail to get state for " + participantID).Error())
	}  
	// else if creditAsBytes == nil {
	// 	logger.Error("CreditRead:  Corrupt reading record ", err.Error())
	// 	return shim.Error(errors.New("CreditRead: Credit does not exist " + participantID).Error())
	// }

	// For log printing credit Information & check whether the credit does exist
	var credit Credit
	err = json.Unmarshal(creditAsBytes, &credit)
	if err != nil {
		logger.Error("CreditRead:  Corrupt reading record ", creditAsBytes)
		return shim.Error(errors.New("CreditRead: Credit does not exist " + string(creditAsBytes)).Error())
	}
	logger.Info("----CreditRead---", credit)
	// For log printing credit Information

	return shim.Success(creditAsBytes)
}

func  (rdg *SmartContract) CreditUpdate(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	//to do
	return shim.Success(nil)

}

func  (rdg *SmartContract) CreditDelete(stub shim.ChaincodeStubInterface, ticketID string) peer.Response {
	//to do
	return shim.Success(nil)

}

// func TicketAdd(stub shim.ChaincodeStubInterface, args []string){
// 	//to do
// 	return shim.Success(nil)
// }

// func TicketDelete(stub shim.ChaincodeStubInterface, ticketID string){
// 	//to do
// 	return shim.Success(nil)
// }

// func TicketUpdate(stub shim.ChaincodeStubInterface, args []string){
// 	//to do
// 	return shim.Success(nil)
// }

// func TicketQuery(stub shim.ChaincodeStubInterface, args []string){
// 	//to do 
// 	return shim.Success(nil)
// }
