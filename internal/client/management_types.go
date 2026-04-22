package client

type ConfigOverlay struct {
	InstanceID         string              `json:"instanceId"`
	Locations          map[string]Location `json:"locations"`
	Users              []User              `json:"users"`
	Endpoints          []Endpoint          `json:"endpoints"`
	ReplicationStreams []ReplicationStream `json:"replicationStreams"`
	Version            int64               `json:"version"`
	UpdatedAt          string              `json:"updatedAt"`
}

type Location struct {
	Name              string           `json:"name"`
	LocationType      string           `json:"locationType"`
	ObjectID          string           `json:"objectId,omitempty"`
	IsBuiltin         bool             `json:"isBuiltin,omitempty"`
	IsTransient       bool             `json:"isTransient,omitempty"`
	IsCold            bool             `json:"isCold,omitempty"`
	LegacyAwsBehavior bool             `json:"legacyAwsBehavior,omitempty"`
	SizeLimitGB       int64            `json:"sizeLimitGB,omitempty"`
	Details           *LocationDetails `json:"details,omitempty"`
}

type LocationDetails struct {
	AccessKey            string   `json:"accessKey,omitempty"`
	SecretKey            string   `json:"secretKey,omitempty"`
	BucketName           string   `json:"bucketName,omitempty"`
	BucketMatch          *bool    `json:"bucketMatch,omitempty"`
	Endpoint             string   `json:"endpoint,omitempty"`
	Region               string   `json:"region,omitempty"`
	ServerSideEncryption *bool    `json:"serverSideEncryption,omitempty"`
	StorageClass         string   `json:"storageClass,omitempty"`
	MpuBucketName        string   `json:"mpuBucketName,omitempty"`
	Username             string   `json:"username,omitempty"`
	Password             string   `json:"password,omitempty"`
	TenantName           string   `json:"tenantName,omitempty"`
	SubscriptionID       string   `json:"subscriptionId,omitempty"`
	ResourceGroup        string   `json:"resourceGroup,omitempty"`
	StorageAccountName   string   `json:"storageAccountName,omitempty"`
	StorageContainerName string   `json:"storageContainerName,omitempty"`
	NsID                 string   `json:"nsId,omitempty"`
	RepoID               []string `json:"repoId,omitempty"`
	ProxyPath            string   `json:"proxyPath,omitempty"`
	BootstrapList        []string `json:"bootstrapList,omitempty"`
	ChordCos             *int64   `json:"chordCos,omitempty"`
	CodingParts          *int64   `json:"codingParts,omitempty"`
	DataParts            *int64   `json:"dataParts,omitempty"`
	GcpEndpoint          string   `json:"gcpEndpoint,omitempty"`
	BucketPrefix         string   `json:"bucketPrefix,omitempty"`
}

type User struct {
	AccountName string `json:"accountName"`
	AccessKey   string `json:"accessKey,omitempty"`
	SecretKey   string `json:"secretKey,omitempty"`
	ARN         string `json:"arn,omitempty"`
	CanonicalID string `json:"canonicalId,omitempty"`
	Email       string `json:"email,omitempty"`
	CreateDate  string `json:"createDate,omitempty"`
	UserName    string `json:"userName,omitempty"`
	ID          string `json:"id,omitempty"`
}

type Endpoint struct {
	Hostname     string `json:"hostname"`
	LocationName string `json:"locationName,omitempty"`
	IsBuiltin    bool   `json:"isBuiltin,omitempty"`
}

type ReplicationStream struct {
	StreamID    string             `json:"streamId,omitempty"`
	Name        string             `json:"name"`
	Version     int64              `json:"version"`
	Enabled     bool               `json:"enabled"`
	Source      *ReplicationSource `json:"source,omitempty"`
	Destination *ReplicationDest   `json:"destination,omitempty"`
}

type ReplicationSource struct {
	BucketName string `json:"bucketName"`
	Prefix     string `json:"prefix"`
	Location   string `json:"location,omitempty"`
}

type ReplicationDest struct {
	BucketName            string                    `json:"bucketName,omitempty"`
	Location              string                    `json:"location,omitempty"`
	Locations             []ReplicationDestLocation `json:"locations,omitempty"`
	PreferredReadLocation string                    `json:"preferredReadLocation,omitempty"`
	Role                  string                    `json:"role,omitempty"`
}

type ReplicationDestLocation struct {
	Name         string `json:"name"`
	StorageClass string `json:"storageClass,omitempty"`
}

type WorkflowFilter struct {
	ObjectKeyPrefix string        `json:"objectKeyPrefix,omitempty"`
	ObjectTags      []WorkflowTag `json:"objectTags,omitempty"`
}

type WorkflowTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type BucketWorkflowExpiration struct {
	WorkflowID                                string          `json:"workflowId,omitempty"`
	Name                                      string          `json:"name,omitempty"`
	Enabled                                   bool            `json:"enabled"`
	BucketName                                string          `json:"bucketName"`
	Type                                      string          `json:"type"`
	Filter                                    *WorkflowFilter `json:"filter,omitempty"`
	CurrentVersionTriggerDelayDate            string          `json:"currentVersionTriggerDelayDate,omitempty"`
	CurrentVersionTriggerDelayDays            *int64          `json:"currentVersionTriggerDelayDays,omitempty"`
	ExpireDeleteMarkersTrigger                *bool           `json:"expireDeleteMarkersTrigger,omitempty"`
	IncompleteMultipartUploadTriggerDelayDays *int64          `json:"incompleteMultipartUploadTriggerDelayDays,omitempty"`
	PreviousVersionTriggerDelayDays           *int64          `json:"previousVersionTriggerDelayDays,omitempty"`
}

type BucketWorkflowTransition struct {
	WorkflowID       string          `json:"workflowId,omitempty"`
	Name             string          `json:"name,omitempty"`
	Enabled          bool            `json:"enabled"`
	BucketName       string          `json:"bucketName"`
	Type             string          `json:"type"`
	Filter           *WorkflowFilter `json:"filter,omitempty"`
	LocationName     string          `json:"locationName"`
	ApplyToVersion   string          `json:"applyToVersion"`
	TriggerDelayDate string          `json:"triggerDelayDate,omitempty"`
	TriggerDelayDays *int64          `json:"triggerDelayDays,omitempty"`
}
