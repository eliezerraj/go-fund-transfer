package listener

import(
	"fmt"
	"math/rand"
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/go-fund-transfer/internal/core"
)

var childLogger = log.With().Str("handler", "listener").Logger()

const(
	lifeSpan = 5 * time.Second
)

type TokenRefresh struct {
	accessToken chan core.TokenSA
	username	string
	password 	string
	hostURL		string
	authorize func() (string, error)
}

func NewToken(ctx context.Context, auth func() (string, error)) *TokenRefresh {
	a := &TokenRefresh{
		accessToken: make(chan core.TokenSA),
		authorize:   auth,
	}
	go a.RefreshToken(ctx)
	return a
}

func (t *TokenRefresh) RefreshToken(ctx context.Context){
	childLogger.Debug().Msg("RefreshJWT")

	var token string
	var err error
	var token_duration time.Duration

	token, _ = t.authorize()
	token_duration = 30 * time.Second
	jwt_expired := time.After(token_duration - lifeSpan)
	
	for {
		select{
			case t.accessToken <- core.TokenSA{Token: token, Err: err}:
			case <-jwt_expired:
				fmt.Println("Token expired")
				token, _ = t.authorize()

				fmt.Println("===> refreshed new token:", token)

				jwt_expired = time.After(token_duration - lifeSpan)
			case <-ctx.Done():
				fmt.Println("Done !!!!!!!")
				return
		}
	}
}

func (t *TokenRefresh) GetToken() (string, error) {
	childLogger.Debug().Msg("GetToken")
	
	res := <-t.accessToken

	fmt.Println("GetToken new token !!!! :", res)

	return res.Token, res.Err
}

func AuthFuncTest() (string, error) {
	childLogger.Debug().Msg("AuthFuncTest")
	
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	token := fmt.Sprint("ABC", r.Intn(100))
	return token, nil
}

func (t *TokenRefresh) AuthFunc() (string, error) {
	childLogger.Debug().Msg("AuthFunc")
	
	res_interface_token, err := s.restApiService.CallRestApi(ctx, "POST", t.hostURL , nil, nil, nil)
	if err != nil {
		return nil, err
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	token := fmt.Sprint("ABC", r.Intn(100))
	return token, nil
}