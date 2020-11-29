package messages

import "encoding/json"

// AsyncCommand does not require a response
type AsyncCommand struct {
	RequestID   string          `json:"request_id"`
	MessageType MessageType     `json:"message_type"`
	RequestType RequestType     `json:"request_type"`
	Payload     json.RawMessage `json:"payload"`
}

// PayloadLoadFile tells agent to download the file into memory and get ready to print
type PayloadLoadFile struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// AgentStatus is the status of the printer
type AgentStatus string

// StatusPrinting means printer is printing
const StatusPrinting AgentStatus = "Printing"

// StatusReady is file loaded and ready to print
const StatusReady AgentStatus = "READY"

// StatusUnknown is the unknown state
const StatusUnknown AgentStatus = "UNKNOWN"

// StatusIdle is the printer waiting
const StatusIdle AgentStatus = "IDLE"

// AgentInfo used for info panel on the front end
type AgentInfo struct {
	Busy   bool        `json:"busy"` // No print commands allowed
	Status AgentStatus `json:"status"`
}

// MessageType shows the type of message
type MessageType string

// TypeCommand tells the recipient if an action is needed
const TypeCommand = "COMMAND"

// TypeInfo tells the recipient if an action is needed
const TypeInfo = "INFO"

// RequestType are just enumerated values to know what to do with the message
type RequestType string

// InfoAgentStatus sends the agent printer struct
const InfoAgentStatus RequestType = "AGENT_STATUS"

// CommandLevelBedTest sends the level bed command
const CommandLevelBedTest RequestType = "LEVEL_BED"

// CommandAutoHome sends the auto home command
const CommandAutoHome RequestType = "AUTO_HOME"

// CommandUnlockPrinter unlocks the printer
const CommandUnlockPrinter RequestType = "UNLOCK_PRINTER"

// CommandLoad will tell the printer to load
const CommandLoad RequestType = "COMMAND_LOAD"

// CommandStart will tell the printer to print
const CommandStart RequestType = "COMMAND_START"

// CommandPause will tell the printer to pause
const CommandPause RequestType = "COMMAND_PAUSE"

// CommandCancel will tell the printer to cancel
const CommandCancel RequestType = "COMMAND_CANCEL"
