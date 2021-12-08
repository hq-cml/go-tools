package errors

import "testing"

func Test_genErr1(t *testing.T) {
    tests := []struct {
        name    string
        wantErr bool
    }{
        {
            name:    "case1",
            wantErr: true,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := genErr1()
            t.Logf("The Error is: %v", err)
        })
    }
}

func Test_genErr2(t *testing.T) {
    tests := []struct {
        name    string
        wantErr bool
    }{
        {
            name:    "case1",
            wantErr: true,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := genErr2()
            t.Logf("The Error is: %+v", err) // 错误栈包裹（打印需要 %+v）
        })
    }
}

func Test_genErr3(t *testing.T) {
    tests := []struct {
        name    string
        wantErr bool
    }{
        {
            name:    "case1",
            wantErr: true,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := genErr3()
            t.Logf("The Error is: %+v", err) // 错误栈包裹（打印需要 %+v）
        })
    }
}