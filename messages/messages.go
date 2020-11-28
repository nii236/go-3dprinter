package messages

// AsyncCommand does not require a response
type AsyncCommand struct {
	RequestID string
	Type      CommandType
	Payload   []byte
}

// CommandType are just enumerated values to know what to do with the message
type CommandType string

// AgentStatus sends the agent printer struct
const AgentStatus CommandType = "AGENT_STATUS"

// LevelBedTest sends the level bed command
const LevelBedTest CommandType = "LEVEL_BED"

// AutoHome sends the auto home command
const AutoHome CommandType = "AUTO_HOME"

// UnlockPrinter unlocks the printer
const UnlockPrinter CommandType = "UNLOCK_PRINTER"

// CommandLoad will tell the printer to load
const CommandLoad CommandType = "COMMAND_LOAD"

// CommandPrint will tell the printer to print
const CommandPrint CommandType = "COMMAND_PRINT"

// CommandPause will tell the printer to pause
const CommandPause CommandType = "COMMAND_PAUSE"

// CommandCancel will tell the printer to cancel
const CommandCancel CommandType = "COMMAND_CANCEL"
