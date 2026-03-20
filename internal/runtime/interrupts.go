package runtime

type Interrupt interface {
	Message() string
}

type InterruptBase struct {
	message string
}

func (i *InterruptBase) Message() string {
	if i.message == "" {
		return "interrupt"
	}
	return i.message
}

type BreakInterrupt struct {
	InterruptBase
}

type ContinueInterrupt struct {
	InterruptBase
}

func NewBreakInterrupt(msg string) *BreakInterrupt {
	return &BreakInterrupt{InterruptBase{message: msg}}
}

func NewContinueInterrupt(msg string) *ContinueInterrupt {
	return &ContinueInterrupt{InterruptBase{message: msg}}
}
