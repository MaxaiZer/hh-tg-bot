package bot

import (
	"encoding/json"
)

type userContext struct {
	chatID          int64
	curCommand      command
	curCommandName  string
	curCommandState []byte
}

func newUserContext(chatID int64) *userContext {
	return &userContext{chatID: chatID}
}

func (u *userContext) RunCommand(command command, name string) {
	u.setCommand(command, name)
	u.curCommand.Run()
}

func (u *userContext) ResumeCommandAfterBotRestart(command command) { //ToDo: what if command starts only after Run()?
	u.setCommand(command, u.curCommandName)
}

func (u *userContext) HasRunningCommand() bool {
	return u.curCommand != nil
}

func (u *userContext) OnUserInput(input string) {
	u.curCommand.OnUserInput(input)
}

func (u *userContext) MarshalJSON() ([]byte, error) {

	var cmdState []byte
	var err error
	if u.curCommand != nil {
		if saveableCmd, ok := u.curCommand.(saveable); ok {
			cmdState, err = saveableCmd.SaveState()
		}
	}
	if err != nil {
		return nil, err
	}

	type Alias userContext
	return json.Marshal(&struct {
		ChatID          int64  `json:"chatID"`
		CurCommandName  string `json:"curCommandName"`
		CurCommandState []byte `json:"curCommandState"`
		*Alias
	}{
		ChatID:          u.chatID,
		CurCommandName:  u.curCommandName,
		CurCommandState: cmdState,
		Alias:           (*Alias)(u),
	})
}

func (u *userContext) UnmarshalJSON(data []byte) error {

	type Alias userContext
	aux := &struct {
		ChatID          int64  `json:"chatID"`
		CurCommandName  string `json:"curCommandName"`
		CurCommandState []byte `json:"curCommandState"`
		*Alias
	}{
		Alias: (*Alias)(u),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	u.chatID = aux.ChatID
	u.curCommandName = aux.CurCommandName
	u.curCommandState = aux.CurCommandState
	return nil
}

func (u *userContext) setCommand(command command, name string) {
	u.curCommand = command
	u.curCommandName = name
	u.curCommand.WithFinishCallback(func() {
		u.curCommand = nil
		u.curCommandName = ""
	})
	u.curCommand.WithKeyboardOnFinalMessage(defaultReplyKeyboard())
}
