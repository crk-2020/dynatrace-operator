package authtoken

import (
	"context"
	"testing"
	"time"

	dynatracev1beta1 "github.com/Dynatrace/dynatrace-operator/src/api/v1beta1"
	"github.com/Dynatrace/dynatrace-operator/src/dtclient"
	"github.com/Dynatrace/dynatrace-operator/src/scheme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testDynakubeName = "test-dynakube"
	testNamespace    = "test-namespace"
	secretName       = testDynakubeName + dynatracev1beta1.AuthTokenSecretSuffix
	testToken        = "dt.testtoken.test"
)

var (
	testAgAuthTokenResponse = &dtclient.ActiveGateAuthTokenInfo{
		TokenId: "test",
		Token:   "dt.some.valuegoeshere",
	}
)

func newTestReconcilerWithInstance(client client.Client) *Reconciler {
	instance := &dynatracev1beta1.DynaKube{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testDynakubeName,
		},
		Spec: dynatracev1beta1.DynaKubeSpec{
			APIURL: "https://testing.dev.dynatracelabs.com/api",
		},
	}
	dtc := &dtclient.MockDynatraceClient{}
	dtc.On("GetActiveGateAuthToken", mock.Anything).Return(testAgAuthTokenResponse, nil)

	r := NewReconciler(client, client, scheme.Scheme, instance, dtc)
	return r
}

func TestReconcile(t *testing.T) {
	t.Run(`reconcile auth token for first time`, func(t *testing.T) {
		r := newTestReconcilerWithInstance(fake.NewClientBuilder().Build())
		update, err := r.Reconcile()
		require.NoError(t, err)

		var authToken corev1.Secret
		_ = r.client.Get(context.TODO(), client.ObjectKey{Name: r.dynakube.ActiveGateAuthTokenSecret(), Namespace: testNamespace}, &authToken)

		assert.NotEmpty(t, authToken.Data[ActiveGateAuthTokenName])
		assert.True(t, update)
	})
	t.Run(`reconcile outdated auth token`, func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:              secretName,
					Namespace:         testNamespace,
					CreationTimestamp: metav1.Time{Time: time.Now().Add(-AuthTokenRotationInterval).Add(-5 * time.Second)},
				},
				Data: map[string][]byte{ActiveGateAuthTokenName: []byte(testToken)},
			}).
			Build()

		r := newTestReconcilerWithInstance(clt)
		update, err := r.Reconcile()
		require.NoError(t, err)

		var authToken corev1.Secret
		_ = r.client.Get(context.TODO(), client.ObjectKey{Name: r.dynakube.ActiveGateAuthTokenSecret(), Namespace: testNamespace}, &authToken)

		assert.NotEqual(t, authToken.Data[ActiveGateAuthTokenName], []byte(testToken))
		assert.True(t, update)
	})
	t.Run(`reconcile valid auth token`, func(t *testing.T) {
		clt := fake.NewClientBuilder().
			WithScheme(scheme.Scheme).
			WithObjects(&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:              secretName,
					Namespace:         testNamespace,
					CreationTimestamp: metav1.Time{Time: time.Now().Add(-AuthTokenRotationInterval).Add(1 * time.Minute)},
				},
				Data: map[string][]byte{ActiveGateAuthTokenName: []byte(testToken)},
			}).
			Build()
		r := newTestReconcilerWithInstance(clt)

		update, err := r.Reconcile()
		require.NoError(t, err)

		var authToken corev1.Secret
		_ = r.client.Get(context.TODO(), client.ObjectKey{Name: r.dynakube.ActiveGateAuthTokenSecret(), Namespace: testNamespace}, &authToken)

		assert.Equal(t, authToken.Data[ActiveGateAuthTokenName], []byte(testToken))
		assert.True(t, update)
	})
}
