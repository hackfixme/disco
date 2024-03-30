package types

type StoreGetRequest struct {
	Key       string
	Namespace string
}

type StoreSetRequest struct {
	Key       string
	Value     []byte
	Namespace string
}

type StoreSetResponse struct {
	*Response
}

type StoreKeysRequest struct {
	Namespace string
	Prefix    string
}

type StoreKeysResponse struct {
	*Response
	Data map[string][]string `json:"keys"`
}
