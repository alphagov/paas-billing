package auth

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"time"

	"golang.org/x/oauth2"

	jwt "github.com/dgrijalva/jwt-go"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type rsaKey map[string]string

var _ = Describe("UAA", func() {
	var (
		mux           *http.ServeMux
		server        *httptest.Server
		fakeTokenKeys []rsaKey

		uaa *UAA
	)

	fixtureRSAKey1 := rsaKey{
		"kty":   "RSA",
		"e":     "AQAB",
		"use":   "sig",
		"kid":   "z8ayf54zpxaljbfrvgn4",
		"alg":   "RS256",
		"value": "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAqEh8c4plYqKkYqxeddTa\naC/8Z6oUiy1TRWv/msuP+5xR4GjoEw2u76dn+sZCwXc1+YT/435wj9LDGFVtFmna\noQcPxpQr5DRjrJlG5CVqxEO1DTJhn2EBZ2xQ/UcCVFB/GRWSq/eP0zT4qyZlBEPR\nxDmTPnrKQqZVTzGMoy0YQGCli58Tlg4asT1zN4gBx7k64dwmgCs5OLcTe8gaajub\ngwN94YDvnikRjh/taQF5+Yil3gzd3O0ZG1coTc5P2MZ9aJd3fU9Yvkq36TWwVME+\nMSV1G18bZdEnKKJI6XZsFvwLVzFfWgZspHvOWmvzDmIH9z3WPKWmBsd/ItoW0kuW\nVwIDAQAB\n-----END PUBLIC KEY-----",
		"n":     "AKhIfHOKZWKipGKsXnXU2mgv_GeqFIstU0Vr_5rLj_ucUeBo6BMNru-nZ_rGQsF3NfmE_-N-cI_SwxhVbRZp2qEHD8aUK-Q0Y6yZRuQlasRDtQ0yYZ9hAWdsUP1HAlRQfxkVkqv3j9M0-KsmZQRD0cQ5kz56ykKmVU8xjKMtGEBgpYufE5YOGrE9czeIAce5OuHcJoArOTi3E3vIGmo7m4MDfeGA754pEY4f7WkBefmIpd4M3dztGRtXKE3OT9jGfWiXd31PWL5Kt-k1sFTBPjEldRtfG2XRJyiiSOl2bBb8C1cxX1oGbKR7zlpr8w5iB_c91jylpgbHfyLaFtJLllc",
	}

	fixturePrivateRSAKey1, _ := jwt.ParseRSAPrivateKeyFromPEM([]byte("-----BEGIN RSA PRIVATE KEY-----\nMIIEogIBAAKCAQEAqEh8c4plYqKkYqxeddTaaC/8Z6oUiy1TRWv/msuP+5xR4Gjo\nEw2u76dn+sZCwXc1+YT/435wj9LDGFVtFmnaoQcPxpQr5DRjrJlG5CVqxEO1DTJh\nn2EBZ2xQ/UcCVFB/GRWSq/eP0zT4qyZlBEPRxDmTPnrKQqZVTzGMoy0YQGCli58T\nlg4asT1zN4gBx7k64dwmgCs5OLcTe8gaajubgwN94YDvnikRjh/taQF5+Yil3gzd\n3O0ZG1coTc5P2MZ9aJd3fU9Yvkq36TWwVME+MSV1G18bZdEnKKJI6XZsFvwLVzFf\nWgZspHvOWmvzDmIH9z3WPKWmBsd/ItoW0kuWVwIDAQABAoH/RF0uMcIHbgqkvXFI\n7pWKJMlZwMNXlTLUoV8+d6Q62fynRoNXxGXKq5FWrInelLnZM4TUb5buI397wmbx\n6ikWqFQ2FHYdXpfp5jRemFCbDyBybOoKvrSp3VojjMFMMPSCra4V58aqpyLd4qm2\nYAUtMooxRzCa+niYL5PxjljDgWMZOL3OMcakCNp3mSckWQj3zyq6pdSTHzb0bW/F\nqeJS+o3YkvMiSjfZhy9duVjqOPFDPp1oR1/52Ji1+m6El5CxwwevOpMauL79uktk\nOuTpC8o5SgUGyVfiPxtQdyrXAlavsqE3UYGRWpPArFYFNHuErL5f7m2Vz03ehyG7\nFgSBAoGBANDGDXYkJ5h5YMkDKgKmkren4RLuPYVrKzTfiG72W/gU/WpsAMDS74Xq\nGJQ0Jndm66MkcVoZfi8I/tQ/4h5oEtu5R1FyNr6Dlc1EZfToZXK74zE0GSK7ooOb\n8RMhBlOQlVtzTbKE/gGzbR8Yzj9g0sWXt7E3vU8y4T8BM5eA4JLZAoGBAM5Zp7PB\nszjWpUahnCyqj5TXL+e3753EN/0Hl8s0uEiwyPgRFJgM0dEwQDluLSErlG6Shje3\nqV04UgjLHDPX4bIcTJajEtv2Jtq1VtK/WPzV/yO1cYCGz3NUGaz9Z/WhCeX0V5RY\nLZ76i/YGUiYJ3D+GQ4+qUOnvP6ft9KE+WFSvAoGBALwMWN28TSoK0oHc5r9CeM8S\nWSprC2EcmetjGQoRv99iUKzGIZuNpA/0PzVnD+rm+oKVdcBZTA5jxN07uZn31lyx\ns6qJ/QN3lLwyyr9hgNdqo4aTTby6U/TFxsybJ46nodCguDB/mCfCDR1Ag64UsWUn\ndl8bPNqUkszkcSsa+61pAoGBAIYZYZDCCpSfeV0DXZjxZsnVZj5yHHgssi3vp0fZ\nhQFIUfJUN0vw2NHXR4WLAi0SQy3wbuT6qEf6d+VbCYLvgq7bETK721+zAeEUA86F\np3D4KQytt4tNELfkKaNwMwU/mE0mk1vGSi+MpzRFO1GZCtcFjBZrGpZMctPRIi8/\ncuvlAoGAEwnvtw+S5gJoSLjkR6tqNM/E4Cy+3LlWJsuFSjLMemttE4fkFbWoGJAl\n9rqRTbFRvuCNTthx4kxjYCOQWsKIseP6EvaWEQpv9jph33irZoVJ4FJVUq6gbyKt\n7AycqeuWX+TuIs0peAehHzVcGHv4ANnyC2mFjyxUkFa+MwawGu8=\n-----END RSA PRIVATE KEY-----\n"))

	fixtureRSAKey2 := map[string]string{
		"kty":   "RSA",
		"e":     "AQAB",
		"use":   "sig",
		"kid":   "sjsrunokb6koje7wab9c",
		"alg":   "RS256",
		"value": "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA9VvLFuB8cTfsTg6n6Pdz\nG6mjgGq6wI9KLT9rqDmpxVGQnn67FZQ7DnItw/d809LDyieqLN23V3I6fwd4FvWh\nnon6cwufE0Isr+YERl3WhDwXCeLnXlUSVpHNuOD/BLAwNYU3OWJTVRbENmtgHZ8q\nJG4LM6m9K903hxLunSFzBHXwspC5kItU7QlZpdF/tcs98BhRmebX8hSvuF+VppvM\nl2w62QUbpWfVDS8wwUKXRHMu8LOtVVp9UO+SLgDEOkLgoYIHRbhlAUNCVkTll5yZ\nUha+iJm8y6gU4CyANXlr9jTGcXAleW9z/oUueElGeopQnSJfPzul8q1pNbfabgzG\nRQIDAQAB\n-----END PUBLIC KEY-----",
		"n":     "APVbyxbgfHE37E4Op-j3cxupo4BqusCPSi0_a6g5qcVRkJ5-uxWUOw5yLcP3fNPSw8onqizdt1dyOn8HeBb1oZ6J-nMLnxNCLK_mBEZd1oQ8Fwni515VElaRzbjg_wSwMDWFNzliU1UWxDZrYB2fKiRuCzOpvSvdN4cS7p0hcwR18LKQuZCLVO0JWaXRf7XLPfAYUZnm1_IUr7hflaabzJdsOtkFG6Vn1Q0vMMFCl0RzLvCzrVVafVDvki4AxDpC4KGCB0W4ZQFDQlZE5ZecmVIWvoiZvMuoFOAsgDV5a_Y0xnFwJXlvc_6FLnhJRnqKUJ0iXz87pfKtaTW32m4MxkU",
	}

	fixturePrivateRSAKey2, _ := jwt.ParseRSAPrivateKeyFromPEM([]byte("-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA9VvLFuB8cTfsTg6n6PdzG6mjgGq6wI9KLT9rqDmpxVGQnn67\nFZQ7DnItw/d809LDyieqLN23V3I6fwd4FvWhnon6cwufE0Isr+YERl3WhDwXCeLn\nXlUSVpHNuOD/BLAwNYU3OWJTVRbENmtgHZ8qJG4LM6m9K903hxLunSFzBHXwspC5\nkItU7QlZpdF/tcs98BhRmebX8hSvuF+VppvMl2w62QUbpWfVDS8wwUKXRHMu8LOt\nVVp9UO+SLgDEOkLgoYIHRbhlAUNCVkTll5yZUha+iJm8y6gU4CyANXlr9jTGcXAl\neW9z/oUueElGeopQnSJfPzul8q1pNbfabgzGRQIDAQABAoIBAGs/x6NlVyAKSOHJ\n6D2eRJOX8F9GyAE54TusGDv9kKcuwx902ARTugjTggvCF69j1q977RgVhnnT9Zvn\nQOgQUKhDOdWmA8/gQjZVPhMgG4/L0GpC483JM+3hZ+JjfzWmajxK0dvkjfaIsBX1\nk5r/IuWvsHfRv134IbiKXwESSPtuUzVXso6/zjTPBWbd/KI9rILE3voR6sM4YZpi\ndhZY0IMJ1ZSD2Dcqpl1em0mpnvoooV3o3pz587LOn3tiX3SZS6kRkzlLHjXbLPQD\noxYxt1ZqZhxh23NK8EIpQdn3W1xuqhPocZO75tpNsxRFcioSz/6RwwGEmPfe3AXu\nyTrEUBUCgYEA/7l4+UWExAYGZf9iwx3bYrHRlBnDd3naiykKl86NEwSTd4p2UttV\nqRxAdeDI2xTzTMvhMfQf9s9efvKyg2UhhluJCp4yTbWYV/+u5DjtyIxq9RYax77B\nnKn7T4XdxYLBjbMkQyIBuHuSrFaYORDrtozbNInWWadnlZvh+GUokasCgYEA9Z92\nPsGpXDL8iJfvLvoXWTKFh4aCwZIovstv+aVF5jZKKzTzhr/gNCUBRqg0PX8nIbQP\noPn5CusaLuO/nMm4hJDdvMK0hRQ7X5cugOStLOemQnBJ91Gioz1Ic5BPnd+7VWNh\n09hbs9wy4elsb2Lfbzyz7RlKN2xmwg1AFMwZ988CgYEA9vDt5xjAqmJ/D0Pc5ToR\nvm6kSXXPkbIz3ioVtp6ZEIJcvRUSSdTQFWvYu3wDubuzbrd6kTiDHV0GjWRkCgpA\no3QFFCHLxcrUgDXBd1WaGQ2vw1hDKBwG7vgeXJ6Sl8Y6jlEtdT6DltiNvKoqeQDj\n/fZrP4LTYOQNXSWYwrs8v90CgYEAuQBcfbQ8LdexYeieHNH92A83h/aGcen2io6M\nTopvdZAamSSO8EWBR4U/yspSXqdw/++xfdwJ+nFODVc5MYy2UBMVEGHOuhWdCsjC\nHA8haJsqHQyaiY+RYkZ8VZ6yeQTVAuGSA5AIshX+tS2toM/l3tDn7IOJ5OjfFPYJ\n+CAqxv0CgYB/71Q/htipU0PHp+u3r0CQe/H8POjdp3335cVhN3sUMRJiMrxHMCBh\n3QHRn3/mIpfF+kGdvvNFNnn4WRJO5DBur6tQZfjDJUtHQr+QudgEN8JUmq709jhN\nVGMBGIU/Pf6yO9dkhoB8/DIKpydffV2hWMnL9MR5qaetO3ydLCVD/w==\n-----END RSA PRIVATE KEY-----\n"))

	fixtureRSAKey3 := map[string]string{
		"kty":   "RSA",
		"e":     "AQAB",
		"use":   "sig",
		"kid":   "sbg2ot938sliwhsqoex5",
		"alg":   "RS256",
		"value": "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAqVl4D/xKlgZmPBNL8qQP\nveYIf8j0b/SboICPuAlXPf0Bc5C/cRbBLaxsZx3daTJI/BHXwt1MZ/HxywyPO+W6\nk/K+7Z8jtNRfiVfRRif3kA9K+t0CgSLrKyogo6eqIXkgxFDuAXH9gWS1Y8tdT1+9\nDdBaWQqx0X/N7xFQKvwQO+EdPmNlAoGQ33x+Hb/OiFCTcQrmPdfzT69PlNCsmfQg\n/vf36MmMFBDAzJmFGvTaXBDPRFENKrlKXL9c6VacZm36sYu+nVBZqRgop7ehRG09\nzJjQDK10xdP4rjjJOfHMWUgJmfFAu2KDk79cOCLZqXmy1s2mmZ0cVqiXqckQqhRz\nNQIDAQAB\n-----END PUBLIC KEY-----",
		"n":     "AKlZeA_8SpYGZjwTS_KkD73mCH_I9G_0m6CAj7gJVz39AXOQv3EWwS2sbGcd3WkySPwR18LdTGfx8csMjzvlupPyvu2fI7TUX4lX0UYn95APSvrdAoEi6ysqIKOnqiF5IMRQ7gFx_YFktWPLXU9fvQ3QWlkKsdF_ze8RUCr8EDvhHT5jZQKBkN98fh2_zohQk3EK5j3X80-vT5TQrJn0IP739-jJjBQQwMyZhRr02lwQz0RRDSq5Sly_XOlWnGZt-rGLvp1QWakYKKe3oURtPcyY0AytdMXT-K44yTnxzFlICZnxQLtig5O_XDgi2al5stbNppmdHFaol6nJEKoUczU",
	}

	fixturePrivateRSAKey3, _ := jwt.ParseRSAPrivateKeyFromPEM([]byte("-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEAt25dehj8PkbHFRMNfy+zTDlfiNY4Qyob3xjmfWULe4XXS3Rs\nW3Z83TliqNq4SQopW7DoTJu2Ra7wIREh+FTvN7939b1X/8DVS/7D+xg2pIJgtUVL\n8HX0wIX77XOIvHPyI8ijfR+xuxng3EVjl9WL4gY4DQyUu/plfrD+IIP6lBfAu+KF\nz3QuZZJHtBwLHm6Bg9CYwldfEA3SCKf7SbXG3Q4RChIH/cV402vnQ+Av0xEnvpmx\nGytX/WPcug8PHQkUbINjXL8tyjTAnnt+R90WFBnaGCYitEKbFqRed0r0397DkvI5\njV2ilRurFt72xWxjVg3tJew+dFlwsOSrF++kAQIDAQABAoIBABjXzbk3oRIeK+Bi\n0DUllLcCHjo+KSiPj27LxIu/H6r/GYWSowpQJeEgYIhV9xeNVMSiVRPrEuilJMiV\ntXAYsL1wJSMXHc/5oenE+24KfXwSXF6wn/RVRWy9uL0UJLTBT04hYmMT49JfUuEC\nVNa/iU53YSgDSDGdXBmohwKIXWupASbBtBYxtpBEL3fQuNnG4iON013AtKypjVe3\nNKICqaIOoizHlgmw8CE9kw5E1/2GzeKikv0zMuCjRCwSozcqIvWTCTS7DlyS/FbH\nVVH2XVz4sC41bxoQk3R86ODQ//JfiEiQE6VceUZEkjn4EpUVfVzH8kM8dVEvSXCQ\nXZekrQECgYEA4YadPTculcCCX12S6d+KEZGXVfMJKNRPNmGds6uhYMMPVT82Q1hC\nXp9eCAtJqLKr9puduXtBEBwtq9ajWbU9tpMIYpCsAKFbRZeM3tC0RsAQw22zZAuG\nW9tVQXCMdBV8RfTx3xfa8YXNEgPAKqbsd5fFETi1Rf2g0IjxuOnFaTkCgYEA0Dec\na1LE0xQyaPj3oexpktUJaimFSZNcBsrJx4ZVl3U+LcrOxAhpUqukElVRPgfbWFHs\n7DW20yH8e+Vqk1HzdAIlyCiXh2Mvv7wu/riydmY0UWtpPzFW6SMSRoI/k7qYLNt7\nVDAOU01vzzlbHle4Q9GDYLzpqJULnUazM3pbeQkCgYEAklxuh1/cl8tL0OBFjApK\n7IP0Fw+XDixbDAvl8Mid/tIYjVZsvN/2kroSqF3K+/SYrX7oqYtX+kCPU0oE0R9S\nYb6iXnVNa0tMlKl5/tCrbo8PUgVLus3P8KUzezizrlKTSENjBUnSCZSwNdTBTezu\n4d5ZQofu/PFRAIUfesYcG+ECgYA2JN/mAKXyBaR+K4+paaKibgd+tcFVOp6JnZ4O\n5l3HftNmcQCHdXB98Og/ZDQ2HzDorJUhb25VRNc1GJk4Ke1W02Ajxnpw2FgIUdUe\no8S0iSs9qOK7bgcdpOMRtrj1n2YG9CQD5mMzQkW66z1IjKL777VsKHPSRL+6bDIZ\nRs4WkQKBgCJinoKrf2GixfCbYKleSkAzp3kBfsE4BCy+bBpJO+dq5xqBxdsKzFhj\nDQo+/M54qVuJvHWIhoyZoEqSmJFTF9/4HZAdzFVW1ApmQ5nNskAlV+RwDpa5hEc9\nUym13rOjdY11DCovZ19B2wiwUYvOwZTaEhqMsdLT2K8UJhO6HOyv\n-----END RSA PRIVATE KEY-----\n"))

	BeforeEach(func() {
		fakeTokenKeys = []rsaKey{
			fixtureRSAKey1,
			fixtureRSAKey2,
		}

		mux = http.NewServeMux()
		server = httptest.NewServer(mux)

		mux.HandleFunc("/token_keys", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			tokenKeysResponse := map[string][]rsaKey{
				"keys": fakeTokenKeys,
			}
			tokenKeysResponseJson, err := json.Marshal(&tokenKeysResponse)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write(tokenKeysResponseJson)
		})

		uaa = &UAA{
			Config: &oauth2.Config{
				Endpoint: oauth2.Endpoint{
					TokenURL: server.URL,
				},
			},
		}

	})

	AfterEach(func() {
		server.Close()
	})

	Describe("ClientAuthorizer", func() {
		Describe("composeClaims()", func() {
			var (
				token *jwt.Token
			)

			BeforeEach(func() {
				token = jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
					"foo": "bar",
					"exp": time.Now().Add(1 * time.Hour).Unix(),
				})
			})

			It("should fail if the token has expired", func() {
				token.Claims.(jwt.MapClaims)["exp"] = time.Now().Add(-1 * time.Hour).Unix()
				token.Header["kid"] = fixtureRSAKey1["kid"]
				tokenString, err := token.SignedString(fixturePrivateRSAKey1)
				Expect(err).ToNot(HaveOccurred())

				authorizer, err := uaa.NewAuthorizer(tokenString)
				Expect(err).ToNot(HaveOccurred())
				clientAuthorizer := authorizer.(*ClientAuthorizer)

				err = clientAuthorizer.composeClaims()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("token expired"))
			})

			It("should not fail if the token is valid for the first key", func() {
				token.Header["kid"] = fixtureRSAKey1["kid"]
				tokenString, err := token.SignedString(fixturePrivateRSAKey1)
				Expect(err).ToNot(HaveOccurred())

				authorizer, err := uaa.NewAuthorizer(tokenString)
				Expect(err).ToNot(HaveOccurred())
				clientAuthorizer := authorizer.(*ClientAuthorizer)

				err = clientAuthorizer.composeClaims()
				Expect(err).ToNot(HaveOccurred())
			})

			It("should not fail if the token is for the second key", func() {
				token.Header["kid"] = fixtureRSAKey2["kid"]
				tokenString, err := token.SignedString(fixturePrivateRSAKey2)
				Expect(err).ToNot(HaveOccurred())

				authorizer, err := uaa.NewAuthorizer(tokenString)
				Expect(err).ToNot(HaveOccurred())
				clientAuthorizer := authorizer.(*ClientAuthorizer)

				err = clientAuthorizer.composeClaims()
				Expect(err).ToNot(HaveOccurred())
			})

			It("should fail if the token is not valid for any of the valid keys", func() {
				token.Header["kid"] = fixtureRSAKey3["kid"]
				tokenString, err := token.SignedString(fixturePrivateRSAKey3)
				Expect(err).ToNot(HaveOccurred())

				authorizer, err := uaa.NewAuthorizer(tokenString)
				Expect(err).ToNot(HaveOccurred())
				clientAuthorizer := authorizer.(*ClientAuthorizer)

				err = clientAuthorizer.composeClaims()
				Expect(err).To(HaveOccurred())
			})

		})

	})
})

// via https://gist.github.com/sdorra/1c95de8cb80da31610d2ad767cd6f251
func generateRSAKey() (publicKeyString, privateKeyString string, err error) {
	reader := rand.Reader
	bitSize := 2048

	key, err := rsa.GenerateKey(reader, bitSize)
	if err != nil {
		return publicKeyString, privateKeyString, err
	}

	buf := new(bytes.Buffer)

	err = pem.Encode(buf, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	if err != nil {
		return publicKeyString, privateKeyString, err
	}

	privateKeyString = buf.String()

	buf = new(bytes.Buffer)

	asn1Bytes, err := asn1.Marshal(key.PublicKey)
	if err != nil {
		return publicKeyString, privateKeyString, err
	}
	err = pem.Encode(buf, &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: asn1Bytes,
	})
	if err != nil {
		return publicKeyString, privateKeyString, err
	}
	publicKeyString = buf.String()

	return publicKeyString, privateKeyString, err
}
