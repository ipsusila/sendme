package sendme_test

import (
	"testing"

	"github.com/ipsusila/sendme"
	"github.com/k0kubun/pp/v3"
	"github.com/stretchr/testify/assert"
)

func TestLoadData(t *testing.T) {
	conf := sendme.Config{
		Delivery: &sendme.DeliveryConfig{
			DataFile: "_testdata/rekap 2nd round.xlsx",
		},
	}
	mc, err := sendme.NewMailDataCollection(&conf)
	assert.NoError(t, err)

	pp.Println(mc)

}

func TestParseAddress(t *testing.T) {
	list := []string{
		"me@example.com",
		"abc@mail.me, a.123@mail.me",
		"test123@example.com; helo@mail.me; test@example.com",
	}

	for _, addr := range list {
		addresses, err := sendme.ParseAddressList(addr)
		assert.NoError(t, err)

		pp.Println(addresses)
	}
}
