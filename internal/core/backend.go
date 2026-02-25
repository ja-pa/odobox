package core

const (
	defaultDBPath      = "./voicemail.db"
	defaultCfgPath     = "./config.ini"
	defaultOCRLanguage = "ces+eng"
)

type Backend struct {
	configPath         string
	mailGatewayFactory MailGatewayFactory
	syncStoreFactory   SyncStoreFactory
	ocrService         OCRService
}

func NewBackend(configPath string) *Backend {
	return &Backend{configPath: configPath}
}

func NewBackendWithDeps(configPath string, deps BackendDeps) *Backend {
	return &Backend{
		configPath:         configPath,
		mailGatewayFactory: deps.MailGatewayFactory,
		syncStoreFactory:   deps.SyncStoreFactory,
		ocrService:         deps.OCRService,
	}
}
