package secret

type GetPublicKeyUsecase struct {
	enc SecretEncryptor
}

func NewGetPublicKeyUsecase(enc SecretEncryptor) *GetPublicKeyUsecase {
	return &GetPublicKeyUsecase{enc: enc}
}

func (uc *GetPublicKeyUsecase) Execute() (keyID, base64PublicKey string) {
	return uc.enc.KeyID(), uc.enc.PublicKeyBase64()
}
