package maps

import (
	"reflect"
	"testing"
)

func Test_markerFromString(t *testing.T) {
	type args struct {
		user string
		s    string
	}
	var tests = []struct {
		name    string
		args    args
		want    *Marker
		wantErr bool
	}{
		{
			name: "ok",
			args: args{
				user: "TontonAo",
				s:    "overworld 152 -257 My super awesome base of doom",
			},
			want: &Marker{
				ID:          "PlayerBase",
				X:           152,
				Y:           64,
				Z:           -257,
				Description: "TontonAo",
				Name:        "My super awesome base of doom",
				Dimension:   "overworld",
			},
			wantErr: false,
		},
		{
			name: "ok with spaces",
			args: args{
				user: "TontonAo",
				s:    " end 152  -257  My super awesome base of doom",
			},
			want: &Marker{
				ID:          "PlayerBase",
				X:           152,
				Y:           64,
				Z:           -257,
				Description: "TontonAo",
				Name:        "My super awesome base of doom",
				Dimension:   "end",
			},
			wantErr: false,
		},
		{
			name: "nok",
			args: args{
				user: "TontonAo",
				s:    "152 -257 My super awesome base of doom",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := markerFromString(tt.args.user, tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("markerFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("markerFromString() got = %v, want %v", got, tt.want)
			}
		})
	}
}
