package main

import (
	"context"
	"log"

	"OdorikCentral/internal/adapters/httpapi"
	imapadapter "OdorikCentral/internal/adapters/imap"
	sqliteadapter "OdorikCentral/internal/adapters/sqlite"
	"OdorikCentral/internal/core"
)

type ListVoicemailsRequest = core.ListVoicemailsRequest
type ListVoicemailsResponse = core.ListVoicemailsResponse
type SyncResponse = core.SyncResponse
type UpdateCheckedResponse = core.UpdateCheckedResponse
type ListSMSMessagesRequest = core.ListSMSMessagesRequest
type ListSMSMessagesResponse = core.ListSMSMessagesResponse
type SettingsResponse = core.SettingsResponse
type PatchSettingsRequest = core.PatchSettingsRequest
type PatchSettingsResponse = core.PatchSettingsResponse
type SendSMSRequest = core.SendSMSRequest
type SendSMSResponse = core.SendSMSResponse
type OdorikBalanceResponse = core.OdorikBalanceResponse
type SMSTemplate = core.SMSTemplate
type CreateSMSTemplateRequest = core.CreateSMSTemplateRequest
type UpdateSMSTemplateRequest = core.UpdateSMSTemplateRequest
type DeleteSMSTemplateResponse = core.DeleteSMSTemplateResponse
type ContactInfo = core.ContactInfo
type ImportVCFRequest = core.ImportVCFRequest
type ImportVCFResponse = core.ImportVCFResponse
type ExportVCFResponse = core.ExportVCFResponse
type CreateContactRequest = core.CreateContactRequest
type UpdateContactRequest = core.UpdateContactRequest
type DeleteContactResponse = core.DeleteContactResponse

type App struct {
	ctx      context.Context
	backend  *core.Backend
	httpAPIS *httpapi.Server
}

func NewApp() *App {
	b := core.NewBackendWithDeps("", core.BackendDeps{
		MailGatewayFactory: imapadapter.NewFactory(),
		SyncStoreFactory:   sqliteadapter.NewFactory(),
	})
	return &App{
		backend:  b,
		httpAPIS: httpapi.New(b),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	if err := a.httpAPIS.Start(); err != nil {
		log.Printf("startup warning: %v", err)
	}
}

func (a *App) shutdown(ctx context.Context) {
	if err := a.httpAPIS.Stop(ctx); err != nil {
		log.Printf("shutdown warning: %v", err)
	}
}

func (a *App) ListVoicemails(req ListVoicemailsRequest) (ListVoicemailsResponse, error) {
	return a.backend.ListVoicemails(req)
}

func (a *App) SyncVoicemails(days int) (SyncResponse, error) {
	return a.backend.Sync(days)
}

func (a *App) SetVoicemailChecked(id int, checked bool) (UpdateCheckedResponse, error) {
	return a.backend.SetVoicemailChecked(id, checked)
}

func (a *App) ListSMSMessages(req ListSMSMessagesRequest) (ListSMSMessagesResponse, error) {
	return a.backend.ListSMSMessages(req)
}

func (a *App) SetSMSMessageChecked(id int, checked bool) (UpdateCheckedResponse, error) {
	return a.backend.SetSMSMessageChecked(id, checked)
}

func (a *App) GetVoicemailAudioDataURL(id int) (string, error) {
	return a.backend.GetVoicemailAudioDataURL(id)
}

func (a *App) GetSettings() (SettingsResponse, error) {
	return a.backend.GetSettings()
}

func (a *App) PatchSettings(req PatchSettingsRequest) (PatchSettingsResponse, error) {
	return a.backend.PatchSettings(req)
}

func (a *App) SendSMS(req SendSMSRequest) (SendSMSResponse, error) {
	return a.backend.SendSMS(req)
}

func (a *App) GetOdorikBalance() (OdorikBalanceResponse, error) {
	return a.backend.GetOdorikBalance()
}

func (a *App) ListSMSTemplates() ([]SMSTemplate, error) {
	return a.backend.ListSMSTemplates()
}

func (a *App) CreateSMSTemplate(req CreateSMSTemplateRequest) (SMSTemplate, error) {
	return a.backend.CreateSMSTemplate(req)
}

func (a *App) UpdateSMSTemplate(req UpdateSMSTemplateRequest) (SMSTemplate, error) {
	return a.backend.UpdateSMSTemplate(req)
}

func (a *App) DeleteSMSTemplate(id int) (DeleteSMSTemplateResponse, error) {
	return a.backend.DeleteSMSTemplate(id)
}

func (a *App) ListContacts() ([]ContactInfo, error) {
	return a.backend.ListContacts()
}

func (a *App) ImportVCF(req ImportVCFRequest) (ImportVCFResponse, error) {
	return a.backend.ImportVCF(req)
}

func (a *App) ExportVCF() (ExportVCFResponse, error) {
	return a.backend.ExportVCF()
}

func (a *App) CreateContact(req CreateContactRequest) (ContactInfo, error) {
	return a.backend.CreateContact(req)
}

func (a *App) UpdateContact(req UpdateContactRequest) (ContactInfo, error) {
	return a.backend.UpdateContact(req)
}

func (a *App) DeleteContact(id int) (DeleteContactResponse, error) {
	return a.backend.DeleteContact(id)
}
