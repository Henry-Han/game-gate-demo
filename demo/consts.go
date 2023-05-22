package demo

const (
	GatewayClientAddr = "127.0.0.1:9900"
	GatewayServerAddr = "127.0.0.1:9901"
)

const (
	ConnectionTimeoutSec = 60
	HeartbeatIntervalSec = 30
)

type PackageType = byte

const (
	Heartbeat PackageType = iota
	ClientToServer
	ServerToClient
	ServerBroadcast
)

type Cmd int32

const (
	C2SOperate   Cmd = 1
	C2SBroadcast Cmd = 2
)

const (
	S2CMessage Cmd = 9999
)
