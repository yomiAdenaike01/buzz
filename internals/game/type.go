package game

type messageType = string
type sentFrom = string
type dest = string

const (
	MessageTypeUserJoined       messageType = "USER_JOINED"
	MessageGameCreatedSuccess   messageType = "GAME_CREATED_SUCCESS"
	MessageTypeGameStateChange  messageType = "GAME_STATE_CHANGED"
	MessageTypeGameAskQuestion  messageType = "ASK_QUESTION"
	MessageGameConnectSuccess   messageType = "GAME_CONNECTED_SUCCESS"
	MessageTypeUserLeft         messageType = "USER_LEFT"
	MessageTypeQuestionComplete messageType = "QUESTION_COMPLETE"
	MessageTypeGameStart        messageType = "START_GAME"
	MessageTypeAnswerQuestion   messageType = "ANSWER_QUESTION"
	MessageTypeTillNextRound    messageType = "COUNTDOWN_TO_NEXT_ROUND"

	SentFromGame  sentFrom = "GAME"
	DestAll       dest     = "*"
	DestNotSender dest     = "NOT_SENDER"
)

type gameMessage struct {
	SentFrom    sentFrom    `json:"sent_from"`
	Dest        dest        `json:"dest"`
	MessageType messageType `json:"message_type"`
	Content     string      `json:"content"`
}
