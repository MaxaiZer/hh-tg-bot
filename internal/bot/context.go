package bot

type userContext struct {
	curCommand command
	chatID     int64
}

func newUserContext(chatID int64) *userContext {
	return &userContext{chatID: chatID}
}

func (u *userContext) RunCommand(command command) {
	u.curCommand = command
	u.curCommand.WithFinishCallback(func() {
		u.curCommand = nil
	})
	u.curCommand.WithKeyboardOnFinalMessage(defaultReplyKeyboard())
	u.curCommand.Run()
}

func (u *userContext) HasRunningCommand() bool {
	return u.curCommand != nil
}

func (u *userContext) OnUserInput(input string) {
	u.curCommand.OnUserInput(input)
}
