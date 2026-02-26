export namespace core {
	
	export class ContactInfo {
	    id: number;
	    full_name: string;
	    phone: string;
	    email: string;
	    org: string;
	    note: string;
	    vcard: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new ContactInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.full_name = source["full_name"];
	        this.phone = source["phone"];
	        this.email = source["email"];
	        this.org = source["org"];
	        this.note = source["note"];
	        this.vcard = source["vcard"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class CreateContactRequest {
	    full_name: string;
	    phone: string;
	    email: string;
	    org: string;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateContactRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.full_name = source["full_name"];
	        this.phone = source["phone"];
	        this.email = source["email"];
	        this.org = source["org"];
	        this.note = source["note"];
	    }
	}
	export class CreateSMSTemplateRequest {
	    name: string;
	    body: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateSMSTemplateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.body = source["body"];
	    }
	}
	export class DeleteContactResponse {
	    status: string;
	    id: number;
	
	    static createFrom(source: any = {}) {
	        return new DeleteContactResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.id = source["id"];
	    }
	}
	export class DeleteSMSTemplateResponse {
	    status: string;
	    id: number;
	
	    static createFrom(source: any = {}) {
	        return new DeleteSMSTemplateResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.id = source["id"];
	    }
	}
	export class ExportVCFResponse {
	    status: string;
	    content: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new ExportVCFResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.content = source["content"];
	        this.count = source["count"];
	    }
	}
	export class ImportVCFRequest {
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new ImportVCFRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.content = source["content"];
	    }
	}
	export class ImportVCFResponse {
	    status: string;
	    imported: number;
	    updated: number;
	    skipped: number;
	    processed: number;
	
	    static createFrom(source: any = {}) {
	        return new ImportVCFResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.imported = source["imported"];
	        this.updated = source["updated"];
	        this.skipped = source["skipped"];
	        this.processed = source["processed"];
	    }
	}
	export class ListSMSHistoryRequest {
	    days: number;
	
	    static createFrom(source: any = {}) {
	        return new ListSMSHistoryRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.days = source["days"];
	    }
	}
	export class SMSHistoryItem {
	    id: number;
	    direction: string;
	    occurred_at: string;
	    counterparty: string;
	    message_text: string;
	    subject: string;
	    sender_id: string;
	    success: boolean;
	    provider_response: string;
	    error_message: string;
	
	    static createFrom(source: any = {}) {
	        return new SMSHistoryItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.direction = source["direction"];
	        this.occurred_at = source["occurred_at"];
	        this.counterparty = source["counterparty"];
	        this.message_text = source["message_text"];
	        this.subject = source["subject"];
	        this.sender_id = source["sender_id"];
	        this.success = source["success"];
	        this.provider_response = source["provider_response"];
	        this.error_message = source["error_message"];
	    }
	}
	export class ListSMSHistoryResponse {
	    items: SMSHistoryItem[];
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new ListSMSHistoryResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], SMSHistoryItem);
	        this.count = source["count"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ListSMSMessagesRequest {
	    days: number;
	    checked: string;
	
	    static createFrom(source: any = {}) {
	        return new ListSMSMessagesRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.days = source["days"];
	        this.checked = source["checked"];
	    }
	}
	export class SMSMessageItem {
	    id: number;
	    date_received: string;
	    subject: string;
	    sender_phone?: string;
	    message_text: string;
	    attachment_text: string;
	    is_checked: boolean;
	    attachment_name: string;
	    pdf_downloaded: boolean;
	    contact?: ContactInfo;
	
	    static createFrom(source: any = {}) {
	        return new SMSMessageItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.date_received = source["date_received"];
	        this.subject = source["subject"];
	        this.sender_phone = source["sender_phone"];
	        this.message_text = source["message_text"];
	        this.attachment_text = source["attachment_text"];
	        this.is_checked = source["is_checked"];
	        this.attachment_name = source["attachment_name"];
	        this.pdf_downloaded = source["pdf_downloaded"];
	        this.contact = this.convertValues(source["contact"], ContactInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ListSMSMessagesResponse {
	    items: SMSMessageItem[];
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new ListSMSMessagesResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], SMSMessageItem);
	        this.count = source["count"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ListVoicemailsRequest {
	    days: number;
	    clean: boolean;
	    checked: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new ListVoicemailsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.days = source["days"];
	        this.clean = source["clean"];
	        this.checked = source["checked"];
	        this.version = source["version"];
	    }
	}
	export class VoicemailItem {
	    id: number;
	    date_received: string;
	    subject: string;
	    caller_phone?: string;
	    message_text: string;
	    is_checked: boolean;
	    attachment_name: string;
	    mp3_downloaded: boolean;
	    audio_duration_s?: number;
	    contact?: ContactInfo;
	
	    static createFrom(source: any = {}) {
	        return new VoicemailItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.date_received = source["date_received"];
	        this.subject = source["subject"];
	        this.caller_phone = source["caller_phone"];
	        this.message_text = source["message_text"];
	        this.is_checked = source["is_checked"];
	        this.attachment_name = source["attachment_name"];
	        this.mp3_downloaded = source["mp3_downloaded"];
	        this.audio_duration_s = source["audio_duration_s"];
	        this.contact = this.convertValues(source["contact"], ContactInfo);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ListVoicemailsResponse {
	    items: VoicemailItem[];
	    count: number;
	    clean: boolean;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new ListVoicemailsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.items = this.convertValues(source["items"], VoicemailItem);
	        this.count = source["count"];
	        this.clean = source["clean"];
	        this.version = source["version"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class OdorikBalanceResponse {
	    status: string;
	    balance: string;
	    currency: string;
	    provider_response: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new OdorikBalanceResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.balance = source["balance"];
	        this.currency = source["currency"];
	        this.provider_response = source["provider_response"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class PatchSettingsRequest {
	    settings: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new PatchSettingsRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.settings = source["settings"];
	    }
	}
	export class PatchSettingsResponse {
	    status: string;
	    settings: Record<string, any>;
	
	    static createFrom(source: any = {}) {
	        return new PatchSettingsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.settings = source["settings"];
	    }
	}
	
	
	export class SMSTemplate {
	    id: number;
	    name: string;
	    body: string;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new SMSTemplate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.body = source["body"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class SendSMSRequest {
	    recipient: string;
	    message: string;
	    sender: string;
	
	    static createFrom(source: any = {}) {
	        return new SendSMSRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.recipient = source["recipient"];
	        this.message = source["message"];
	        this.sender = source["sender"];
	    }
	}
	export class SendSMSResponse {
	    status: string;
	    recipient: string;
	    sender: string;
	    encoding: string;
	    chars_used: number;
	    max_single_chars: number;
	    provider_response: string;
	    sent_at: string;
	
	    static createFrom(source: any = {}) {
	        return new SendSMSResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.recipient = source["recipient"];
	        this.sender = source["sender"];
	        this.encoding = source["encoding"];
	        this.chars_used = source["chars_used"];
	        this.max_single_chars = source["max_single_chars"];
	        this.provider_response = source["provider_response"];
	        this.sent_at = source["sent_at"];
	    }
	}
	export class SettingsResponse {
	    settings: Record<string, any>;
	    editable_sections: string[];
	
	    static createFrom(source: any = {}) {
	        return new SettingsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.settings = source["settings"];
	        this.editable_sections = source["editable_sections"];
	    }
	}
	export class SyncResponse {
	    status: string;
	    days: number;
	    stored: number;
	    skipped_duplicates: number;
	    voicemail_stored: number;
	    sms_stored: number;
	    voicemail_skipped: number;
	    sms_skipped: number;
	
	    static createFrom(source: any = {}) {
	        return new SyncResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.days = source["days"];
	        this.stored = source["stored"];
	        this.skipped_duplicates = source["skipped_duplicates"];
	        this.voicemail_stored = source["voicemail_stored"];
	        this.sms_stored = source["sms_stored"];
	        this.voicemail_skipped = source["voicemail_skipped"];
	        this.sms_skipped = source["sms_skipped"];
	    }
	}
	export class UpdateCheckedResponse {
	    status: string;
	    id: number;
	    is_checked: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateCheckedResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.id = source["id"];
	        this.is_checked = source["is_checked"];
	    }
	}
	export class UpdateContactRequest {
	    id: number;
	    full_name: string;
	    phone: string;
	    email: string;
	    org: string;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateContactRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.full_name = source["full_name"];
	        this.phone = source["phone"];
	        this.email = source["email"];
	        this.org = source["org"];
	        this.note = source["note"];
	    }
	}
	export class UpdateSMSTemplateRequest {
	    id: number;
	    name: string;
	    body: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateSMSTemplateRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.body = source["body"];
	    }
	}

}

