// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	apperrors "github.com/kyma-project/control-plane/components/provisioner/internal/apperrors"
	mock "github.com/stretchr/testify/mock"
)

// KubernetesVersionProvider is an autogenerated mock type for the KubernetesVersionProvider type
type KubernetesVersionProvider struct {
	mock.Mock
}

// Get provides a mock function with given fields: runtimeID, tenant
func (_m *KubernetesVersionProvider) Get(runtimeID string, tenant string) (string, apperrors.AppError) {
	ret := _m.Called(runtimeID, tenant)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, string) string); ok {
		r0 = rf(runtimeID, tenant)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 apperrors.AppError
	if rf, ok := ret.Get(1).(func(string, string) apperrors.AppError); ok {
		r1 = rf(runtimeID, tenant)
	} else {
		if ret.Get(1) != nil {
			r1 = ret.Get(1).(apperrors.AppError)
		}
	}

	return r0, r1
}
