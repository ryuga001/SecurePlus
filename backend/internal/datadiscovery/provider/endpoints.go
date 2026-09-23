package provider

import "time"

const (
	EntraTokenURLFormat = "https://login.microsoftonline.com/%s/oauth2/v2.0/token"
	GraphScope          = "https://graph.microsoft.com/.default"
	GraphBaseURL        = "https://graph.microsoft.com/v1.0"

	AzureStorageScope      = "https://storage.azure.com/.default"
	AzureBlobURLFormat     = "https://%s.blob.core.windows.net"
	AzureBlobListQuery     = "?comp=list&maxresults=1"
	AzureStorageAPIVersion = "2021-12-02"

	GoogleTokenURL     = "https://oauth2.googleapis.com/token"
	GoogleDriveScope   = "https://www.googleapis.com/auth/drive.readonly"
	GoogleDriveBaseURL = "https://www.googleapis.com/drive/v3"
	GoogleDriveAbout   = "/about?fields=user(emailAddress)"
	GoogleGrantType    = "urn:ietf:params:oauth:grant-type:jwt-bearer"
	GoogleAssertionTTL = time.Hour
)
