

peer chaincode invoke -n mycc -c '{"Args":["CreditCreate", "123", "500"]}' -C myc
peer chaincode invoke -n mycc -c '{"Args": ["CreditRead", "123"]}' -C myc

peer chaincode invoke -n mycc -c '{"Args":["CreditUpdate", "123", "500", "1"]}' -C myc
peer chaincode invoke -n mycc -c '{"Args":["CreditUpdate", "123", "-50", "2"]}' -C myc
peer chaincode invoke -n mycc -c '{"Args":["CreditUpdate", "123", "30", "3"]}' -C myc

peer chaincode invoke -n mycc -c '{"Args": ["CreditDelete", "123"]}' -C myc

peer chaincode invoke -n mycc -c '{"Args": ["CreditRead", "123"]}' -C myc



peer chaincode invoke -n mycc -c '{"Args":["TicketCreate", ""]}' -C myc
peer chaincode invoke -n mycc -c '{"Args": ["TicketRead", "ticket_1"]}' -C myc
peer chaincode invoke -n mycc -c '{"Args": ["TicketDelete", "ticket_1"]}' -C myc
peer chaincode invoke -n mycc -c '{"Args": ["TicketUpdate", "ticket_1"]}' -C myc


peer chaincode invoke -n mycc -c '{"Args":["TicketCreate","{\"Ticket_TicketID\": \"xxx1\", \"Ticket_Status\":0,\"Ticket_Title\": \"The test Ticket\", \"Ticket_Type\":0, \"Ticket_Value\": 100, \"Ticket_UserID\":\"1\", \"Ticket_Comment\":\"test comment\",\"Ticket_Policy\":\"policy\"}"]}' -C myc

peer chaincode invoke -n mycc -c '{"Args":["TicketCreate","{\"Ticket_TicketID\": \"1\"}"]}' -C 

peer chaincode invoke -n mycc -c '{"Args":["TicketUpdate", "1", "{\"Ticket_TicketID\": \"xxx1\", \"Ticket_Status\":1,\"Ticket_Title\": \"The update Ticket\", \"Ticket_Type\":2, \"Ticket_Value\": 50, \"Ticket_UserID\":\"1\", \"Ticket_Comment\":\"test comment\",\"Ticket_Policy\":\"policy\"}"]}' -C myc

peer chaincode invoke -n mycc -c '{"Args": ["TicketRead", "xxx1"]}' -C myc
