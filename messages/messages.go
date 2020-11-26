package messages

// Command the agent
type Command struct {
	Type    string
	Payload []byte
}

// TypeLevelBedTest sends the level bed command
const TypeLevelBedTest = "LEVEL_BED"

// TypeAutoHome sends the auto home command
const TypeAutoHome = "AUTO_HOME"

// TypePrintLink sends the print command
const TypePrintLink = "PRINT_LINK"

// TypeUnlockPrinter unlocks the printer
const TypeUnlockPrinter = "UNLOCK_PRINTER"
