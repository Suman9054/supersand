package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"log/slog"

	"github.com/biscuit-auth/biscuit-go/v2"
	"github.com/biscuit-auth/biscuit-go/v2/parser"
)


func SetupAuth()(ed25519.PrivateKey,ed25519.PublicKey,error){
  rng:=rand.Reader
	publickey,privatekey,err:=ed25519.GenerateKey(rng)

	if err !=nil{
		slog.Error("error in auth token setup",err)
		return nil,nil,err
	}

	return privatekey,publickey,nil
}


func CreatSessionToken(privatekey ed25519.PrivateKey)(string,error){
  
	var def string
	authority,err:=parser.FromStringBlockWithParams(`
	right("config",{read});
	right("workdir",{read});
	right("workdir",{write})

	`,map[string]biscuit.Term{"read":biscuit.String("read"),"write":biscuit.String("write")})
	
  if err != nil {
		return def,err
	}

	builder:= biscuit.NewBuilder(privatekey)

	builder.AddBlock(authority)
	b,err:= builder.Build()
	if err != nil{
		return def,err
	}

	token,err:=b.Serialize()
	if err != nil{
		return def,err
	}
 
	return base64.RawURLEncoding.EncodeToString(token),nil
}



func Authorizetoken(token string,pubkey ed25519.PublicKey)(biscuit.Authorizer,error){
	raw,err:=base64.RawStdEncoding.DecodeString(token)

	if err != nil{
		return nil,err
	}

	t,err:=biscuit.Unmarshal(raw)

	if err!=nil{
		return nil,err
	}

	authorize,err:=t.Authorizer(pubkey)

	if err != nil{
		return nil,err
	}

	return authorize,nil
}



