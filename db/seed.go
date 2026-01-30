package db

import (
	"Cx_Mcdean_Backend/models"
	"encoding/json"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 文件需求结构
type fileRequirement struct {
	Type string `json:"type"`
	Min  int64  `json:"min"`
}

// 步骤 requirements
type stepRequirements struct {
	Files   []fileRequirement `json:"files,omitempty"`
	Fields  []string          `json:"fields,omitempty"`
	Confirm bool              `json:"confirm,omitempty"`
}

// SeedSubjectSteps 根据模板为每个 subject 生成步骤，并按启用的步骤动态分配 StepOrder
func SeedSubjectSteps(db *gorm.DB) error {
	type tmpl struct {
		Key        string
		Label      string
		DefaultReq stepRequirements
	}

	// templates 的顺序就是默认顺序（用于给启用的步骤分配 StepOrder）
	templates := []tmpl{
		{Key: "received_off_site", Label: "Received Off Site"},
		{Key: "neta_complete", Label: "NETA Complete", DefaultReq: stepRequirements{Files: []fileRequirement{{Type: "test_report", Min: 1}}}},
		{Key: "ship_to_site", Label: "Ship to Site"},
		{Key: "received_on_site", Label: "Received on Site"},
		{Key: "field_install_inspection", Label: "Field Install Inspection", DefaultReq: stepRequirements{Files: []fileRequirement{{Type: "other", Min: 1}}}},
		{Key: "dlro_tested", Label: "Termination DLRO Tested", DefaultReq: stepRequirements{Files: []fileRequirement{{Type: "test_report", Min: 1}}}},
		{Key: "energized", Label: "Energized", DefaultReq: stepRequirements{Confirm: true, Fields: []string{"energized"}}},

		// Panelboard / Generator specific keys（放在合适位置以决定默认顺序）
		{Key: "installed", Label: "Installed", DefaultReq: stepRequirements{Files: []fileRequirement{{Type: "panel_schedule", Min: 1}}}},
		{Key: "terminated", Label: "Terminated", DefaultReq: stepRequirements{Fields: []string{"comments"}}},
		{Key: "tested", Label: "Tested"},
		{Key: "set_in_place", Label: "Set in Place"},
		{Key: "fuel_oil_ready", Label: "Fuel/Oil Ready", DefaultReq: stepRequirements{Fields: []string{"comments"}}},
		{Key: "start_up", Label: "Start-up"},
		{Key: "load_bank", Label: "Load Bank Test", DefaultReq: stepRequirements{Files: []fileRequirement{{Type: "test_report", Min: 1}}}},
	}

	// 每个 subject 的配置：列出要禁用的 key（SkipKeys）和 requirements 覆盖
	type subjectCfg struct {
		Name string
		// 若 key 在这里列出则表示该 subject 不需要该步骤（直接跳过，不写入 DB）
		SkipKeys map[string]bool
		// 针对某些 key 的 requirement 覆盖（优先级高于模板 DefaultReq）
		ReqOverrides map[string]stepRequirements
	}

	subjects := []subjectCfg{
		{
			Name:     "ATS",
			SkipKeys: map[string]bool{
				// 如果 ATS 有需要跳过的步骤，写在这里
			},
			ReqOverrides: map[string]stepRequirements{
				// ATS-specific overrides（如果需要）
			},
		},
		{
			Name: "Panelboard",
			SkipKeys: map[string]bool{
				"neta_complete": true, // Panelboard 不需要 NETA
			},
			ReqOverrides: map[string]stepRequirements{
				"installed":  {Files: []fileRequirement{{Type: "panel_schedule", Min: 1}}},
				"terminated": {Fields: []string{"comments"}},
				"energized":  {Confirm: true, Fields: []string{"energized"}},
			},
		},
		{
			Name: "Generator",
			SkipKeys: map[string]bool{
				"neta_complete":            true,
				"field_install_inspection": true,
			},
			ReqOverrides: map[string]stepRequirements{
				"fuel_oil_ready": {Fields: []string{"comments"}},
				"load_bank":      {Files: []fileRequirement{{Type: "test_report", Min: 1}}},
				"energized":      {Confirm: true, Fields: []string{"energized"}},
			},
		},
	}

	// 逐个 subject 生成步骤，动态分配 StepOrder（只对启用的步骤计数）
	for _, subj := range subjects {
		stepOrderCounter := 0
		for _, t := range templates {
			// 跳过 subject 指定不需要的步骤
			if subj.SkipKeys != nil && subj.SkipKeys[t.Key] {
				continue
			}

			stepOrderCounter++ // 仅对启用的步骤计数
			isActive := true

			// 决定 requirement：优先使用 subject 覆盖，其次模板 default
			var req stepRequirements
			if ro, ok := subj.ReqOverrides[t.Key]; ok {
				req = ro
			} else {
				req = t.DefaultReq
			}

			step := models.DeviceSubjectStep{
				Subject:      subj.Name,
				Key:          t.Key,
				Label:        t.Label,
				StepOrder:    stepOrderCounter,
				IsActive:     isActive,
				Requirements: mustRequirements(req),
			}

			if err := db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "subject"}, {Name: "key"}},
				DoNothing: true,
			}).Create(&step).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

// isEmptyRequirements 显式判断是否为空 requirements（避免使用不可比较的结构体比较）
func isEmptyRequirements(req stepRequirements) bool {
	if req.Confirm {
		return false
	}
	if len(req.Files) != 0 {
		return false
	}
	if len(req.Fields) != 0 {
		return false
	}
	return true
}

// mustRequirements 将 stepRequirements 编码为 datatypes.JSON；若为空则返回 nil（表示 DB 中为 NULL）
func mustRequirements(req stepRequirements) datatypes.JSON {
	if isEmptyRequirements(req) {
		return nil
	}
	raw, err := json.Marshal(req)
	if err != nil {
		// 若序列化失败，返回 nil（你也可以选择 panic 或返回错误，但在 seed 中通常安静跳过）
		return nil
	}
	return datatypes.JSON(raw)
}
