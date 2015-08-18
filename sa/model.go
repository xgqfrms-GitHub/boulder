package sa

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	jose "github.com/letsencrypt/boulder/Godeps/_workspace/src/github.com/letsencrypt/go-jose"
	"github.com/letsencrypt/boulder/core"
)

// regModel is the description of a core.Registration in the database.
type regModel struct {
	ID        int64           `db:"id"`
	Key       []byte          `db:"jwk"`
	KeySHA256 string          `db:"jwk_sha256"`
	Contact   []*core.AcmeURL `db:"contact"`
	Agreement string          `db:"agreement"`
	LockCol   int64
}

// challModel is the description of a core.Challenge in the database
type challModel struct {
	ID              int64  `db:"id"`
	AuthorizationID string `db:"authorizationID"`

	Type       string          `db:"type"`
	Status     core.AcmeStatus `db:"status"`
	Error      []byte          `db:"error"`
	Validated  *time.Time      `db:"validated"`
	URI        *core.AcmeURL   `db:"uri"`
	Token      string          `db:"token"`
	TLS        *bool           `db:"tls"`
	Validation []byte          `db:"validation"`

	LockCol int64
}

// newReg creates a reg model object from a core.Registration
func registrationToModel(r *core.Registration) (*regModel, error) {
	key, err := json.Marshal(r.Key)
	if err != nil {
		return nil, err
	}

	sha, err := core.KeyDigest(r.Key)
	if err != nil {
		return nil, err
	}
	rm := &regModel{
		ID:        r.ID,
		Key:       key,
		KeySHA256: sha,
		Contact:   r.Contact,
		Agreement: r.Agreement,
	}
	return rm, nil
}

func modelToRegistration(rm *regModel) (core.Registration, error) {
	k := &jose.JsonWebKey{}
	err := json.Unmarshal(rm.Key, k)
	if err != nil {
		err = fmt.Errorf("unable to unmarshal JsonWebKey in db: %s", err)
		return core.Registration{}, err
	}
	r := core.Registration{
		ID:        rm.ID,
		Key:       *k,
		Contact:   rm.Contact,
		Agreement: rm.Agreement,
	}
	return r, nil
}

func challengeToModel(c *core.Challenge, authID string) (*challModel, error) {
	cm := challModel{
		AuthorizationID: authID,
		Type:            c.Type,
		Status:          c.Status,
		Validated:       c.Validated,
		URI:             c.URI,
		Token:           c.Token,
		TLS:             c.TLS,
		// Validation:      []byte(c.Validation.FullSerialize()),
	}
	if c.Validation != nil {
		cm.Validation = []byte(c.Validation.FullSerialize())
		if len(cm.Validation) > int(math.Pow(2, 24)) {
			return nil, fmt.Errorf("Validation object is too large to store in the database")
		}
	}
	if c.Error != nil {
		errJSON, err := json.Marshal(c.Error)
		if err != nil {
			return nil, err
		}
		cm.Error = errJSON
	}
	if cm.URI != nil && len(cm.URI.String()) > 255 {
		return nil, fmt.Errorf("URI is too long")
	}
	return &cm, nil
}

func modelToChallenge(cm *challModel) (core.Challenge, error) {
	var val *jose.JsonWebSignature
	var problem core.ProblemDetails
	var err error
	if len(cm.Validation) > 0 {
		val, err = jose.ParseSigned(string(cm.Validation))
		if err != nil {
			return core.Challenge{}, err
		}
	}
	if len(cm.Error) > 0 {
		err := json.Unmarshal(cm.Error, &problem)
		if err != nil {
			return core.Challenge{}, err
		}
	}
	return core.Challenge{
		Type:       cm.Type,
		Status:     cm.Status,
		Error:      &problem,
		Validated:  cm.Validated,
		URI:        cm.URI,
		Token:      cm.Token,
		TLS:        cm.TLS,
		Validation: val,
	}, nil
}
