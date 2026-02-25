package core

type BackendDeps struct {
	MailGatewayFactory MailGatewayFactory
	SyncStoreFactory   SyncStoreFactory
	OCRService         OCRService
}
