package db

import (
	"Cx_Mcdean_Backend/models"
	"encoding/json"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type fileRequirement struct {
	Type string `json:"type"`
	Min  int64  `json:"min"`
}

type stepRequirements struct {
	Files   []fileRequirement `json:"files,omitempty"`
	Fields  []string          `json:"fields,omitempty"`
	Confirm bool              `json:"confirm,omitempty"`
}

func SeedSubjectSteps(db *gorm.DB) error {
	steps := []models.DeviceSubjectStep{
		// ATS
		{Subject: "ATS", Key: "received_off_site", Label: "Received Off Site", StepOrder: 1, IsActive: true},
		{
			Subject:      "ATS",
			Key:          "neta_complete",
			Label:        "NETA Complete",
			StepOrder:    2,
			IsActive:     true,
			Requirements: mustRequirements(stepRequirements{Files: []fileRequirement{{Type: "test_report", Min: 1}}}),
		},
		{Subject: "ATS", Key: "ship_to_site", Label: "Ship to Site", StepOrder: 3, IsActive: true},
		{Subject: "ATS", Key: "received_on_site", Label: "Received on Site", StepOrder: 4, IsActive: true},
		{
			Subject:      "ATS",
			Key:          "field_install_inspection",
			Label:        "Field Install Inspection",
			StepOrder:    5,
			IsActive:     true,
			Requirements: mustRequirements(stepRequirements{Files: []fileRequirement{{Type: "other", Min: 1}}}),
		},
		{
			Subject:      "ATS",
			Key:          "dlro_tested",
			Label:        "Termination DLRO Tested",
			StepOrder:    6,
			IsActive:     true,
			Requirements: mustRequirements(stepRequirements{Files: []fileRequirement{{Type: "test_report", Min: 1}}}),
		},
		{
			Subject:      "ATS",
			Key:          "energized",
			Label:        "Energized",
			StepOrder:    7,
			IsActive:     true,
			Requirements: mustRequirements(stepRequirements{Confirm: true, Fields: []string{"energized"}}),
		},

		// Panelboard
		{Subject: "Panelboard", Key: "received_on_site", Label: "Received on Site", StepOrder: 1, IsActive: true},
		{
			Subject:      "Panelboard",
			Key:          "installed",
			Label:        "Installed",
			StepOrder:    2,
			IsActive:     true,
			Requirements: mustRequirements(stepRequirements{Files: []fileRequirement{{Type: "panel_schedule", Min: 1}}}),
		},
		{
			Subject:      "Panelboard",
			Key:          "terminated",
			Label:        "Terminated",
			StepOrder:    3,
			IsActive:     true,
			Requirements: mustRequirements(stepRequirements{Fields: []string{"comments"}}),
		},
		{Subject: "Panelboard", Key: "tested", Label: "Tested", StepOrder: 4, IsActive: true},
		{
			Subject:      "Panelboard",
			Key:          "energized",
			Label:        "Energized",
			StepOrder:    5,
			IsActive:     true,
			Requirements: mustRequirements(stepRequirements{Confirm: true, Fields: []string{"energized"}}),
		},

		// Generator
		{Subject: "Generator", Key: "received_on_site", Label: "Received on Site", StepOrder: 1, IsActive: true},
		{Subject: "Generator", Key: "set_in_place", Label: "Set in Place", StepOrder: 2, IsActive: true},
		{
			Subject:      "Generator",
			Key:          "fuel_oil_ready",
			Label:        "Fuel/Oil Ready",
			StepOrder:    3,
			IsActive:     true,
			Requirements: mustRequirements(stepRequirements{Fields: []string{"comments"}}),
		},
		{Subject: "Generator", Key: "start_up", Label: "Start-up", StepOrder: 4, IsActive: true},
		{
			Subject:      "Generator",
			Key:          "load_bank",
			Label:        "Load Bank Test",
			StepOrder:    5,
			IsActive:     true,
			Requirements: mustRequirements(stepRequirements{Files: []fileRequirement{{Type: "test_report", Min: 1}}}),
		},
		{
			Subject:      "Generator",
			Key:          "energized",
			Label:        "Energized",
			StepOrder:    6,
			IsActive:     true,
			Requirements: mustRequirements(stepRequirements{Confirm: true, Fields: []string{"energized"}}),
		},
	}

	for _, step := range steps {
		if err := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "subject"}, {Name: "key"}},
			DoNothing: true,
		}).Create(&step).Error; err != nil {
			return err
		}
	}
	return nil
}

func mustRequirements(req stepRequirements) datatypes.JSON {
	if req == (stepRequirements{}) {
		return nil
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return nil
	}
	return datatypes.JSON(raw)
}
