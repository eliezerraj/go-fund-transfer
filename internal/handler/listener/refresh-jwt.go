package listener

import (
	"context"
	"fmt"
	"math/rand"
	"time"
	"encoding/json"

	"github.com/rs/zerolog/log"
	"github.com/go-fund-transfer/internal/core"
)

var childLogger = log.With().Str("handler", "listener").Logger()

const(
	lifeSpan = 100 * time.Second
)

type TokenRefresh struct {
	accessToken chan core.TokenSA
	restApiCallData	core.RestApiCallData
	authorize func() (string, error)
	refreshToken func(context.Context, core.RestApiCallData, interface{}) (interface{}, error)
}

func NewToken(	ctx	context.Context,
				auth func() (string, error), 
				restApiCallData	core.RestApiCallData, 
		      	refreshToken func(context.Context, core.RestApiCallData, interface{}) (interface{}, error)) *TokenRefresh {

	a := &TokenRefresh{	accessToken: make(chan core.TokenSA),
						authorize:   auth,
						restApiCallData: restApiCallData,
						refreshToken: refreshToken,
						}

	go a.RefreshToken(ctx)
	return a
}

func (t *TokenRefresh) RefreshToken(ctx context.Context){
	childLogger.Debug().Msg("RefreshJWT")

	var token string
	var err error
	var token_duration time.Duration

	// t.restApiCallData = payload user/password
	res_interface, err := t.refreshToken(ctx, t.restApiCallData, t.restApiCallData)
	if err != nil {
		childLogger.Error().Err(err).Msg("error parse interface")
	}
	token_parsed := ConvertToToken(res_interface)
	fmt.Println("====> token_parsed:", token_parsed.Token)

	// Test
	token, _ = t.authorize()

	token_duration = 500 * time.Second
	jwt_expired := time.After(token_duration - lifeSpan)
	
	for {
		select{
			case t.accessToken <- core.TokenSA{Token: token, Err: err}:
			case <-jwt_expired:
				fmt.Println("Token expired")

				// t.restApiCallData = payload user/password
				res_interface, err := t.refreshToken(ctx, t.restApiCallData, t.restApiCallData)
				if err != nil {
					childLogger.Error().Err(err).Msg("error parse interface")
				}
				token_parsed := ConvertToToken(res_interface)
				fmt.Println("====> refreshed token_parsed:", token_parsed.Token)
				
				token, _ = t.authorize()

				fmt.Println("===> refreshed new token:", token)

				jwt_expired = time.After(token_duration - lifeSpan)
			case <-ctx.Done():
				fmt.Println("Done !!!!!!!")
				return
		}
	}
}

func ConvertToToken(res_interface interface{}) core.TokenSA {
	childLogger.Debug().Msg("ConvertToToken")

	jsonString, err  := json.Marshal(res_interface)
	if err != nil {
		childLogger.Error().Err(err).Msg("error Marshal")
    }
	var token_parsed core.TokenSA
	json.Unmarshal(jsonString, &token_parsed)

	return token_parsed
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