package seed

import (
	"go-3dprint/agent"
	"go-3dprint/db"

	"github.com/ninja-software/terror"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"syreclabs.com/go/faker"
)

// Run seed funcs
func Run() error {
	for i := 0; i < 10; i++ {
		fname := faker.Company().Bs()
		blob := &db.Blob{Data: []byte(agent.GCodeLevelBedTest), FileName: fname, FileSizeBytes: int64(len([]byte(agent.GCodeLevelBedTest)))}
		err := blob.InsertG(boil.Infer())
		if err != nil {
			return terror.New(err, "")
		}
		gcode := &db.Gcode{
			Name:   fname,
			BlobID: blob.ID,
		}
		err = gcode.InsertG(boil.Infer())
		if err != nil {
			return terror.New(err, "")
		}
	}
	return nil
}
