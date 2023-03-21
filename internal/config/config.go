package config

import (
	"encoding/json"
	"errors"
	"os"
	"path"
)

type Config struct {
	LogLevel  string           `json:"logLevel,omitempty"`
	Endpoints *ConfigEndpoints `json:"endpoints,omitempty"`
	Protocols *ConfigProtocols `json:"protocols,omitempty"`
	Keys      *ConfigKeys      `json:"keys,omitempty"`
}

type ConfigEndpoints struct {
	MainEndpoint string              `json:"mainEndpoint,omitempty"`
	MainProtocol Protocol            `json:"mainProtocol,omitempty"`
	Mapping      map[Protocol]string `json:"mapping,omitempty"`
}

type ConfigProtocols struct {
	BaseProtocol Protocol            `json:"baseProtocol,omitempty"`
	Mapping      map[Protocol]string `json:"mapping,omitempty"`
}

type ConfigKeys struct {
	SharedKey  string            `json:"sharedKey,omitempty"`
	ServerKey  string            `json:"serverKey,omitempty"`
	ClientKeys map[uint32]string `json:"clientKeys,omitempty"`
}

func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	c := new(Config)
	d := json.NewDecoder(f)
	if err := d.Decode(c); err != nil {
		return nil, err
	}
	if c.Endpoints == nil {
		return nil, errors.New("no endpoint configured")
	}
	if c.Protocols == nil {
		return nil, errors.New("no protocol configured")
	}
	if c.Keys == nil {
		return nil, errors.New("no key configured")
	}
	return c, nil
}

var DefaultConfig = &Config{
	Endpoints: &ConfigEndpoints{
		MainEndpoint: "{{ UPSTREAM_ADDRESS }}",
		MainProtocol: ProtocolMajor3Minor2,
		Mapping: map[Protocol]string{
			"{{ CLIENT_VERSION }}": "{{ ENDPOINT_LISTEN_ADDRESS }}",
		},
	},
	Protocols: &ConfigProtocols{
		BaseProtocol: ProtocolMajor3Minor2,
		Mapping: map[Protocol]string{
			ProtocolMajor3Minor2: path.Join("data/mapping/", string(ProtocolMajor3Minor2)),
		},
	},
	Keys: defaultConfigKeys,
}

var defaultConfigKeys = &ConfigKeys{
	SharedKey: "{{ FIRST_PACKET_ENCRYPTION_KEY }}",
	ServerKey: `-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAxbbx2m1feHyrQ7jP+8mtDF/pyYLrJWKWAdEv3wZrOtjOZzeL
GPzsmkcgncgoRhX4dT+1itSMR9j9m0/OwsH2UoF6U32LxCOQWQD1AMgIZjAkJeJv
FTrtn8fMQ1701CkbaLTVIjRMlTw8kNXvNA/A9UatoiDmi4TFG6mrxTKZpIcTInvP
EpkK2A7Qsp1E4skFK8jmysy7uRhMaYHtPTsBvxP0zn3lhKB3W+HTqpneewXWHjCD
fL7Nbby91jbz5EKPZXWLuhXIvR1Cu4tiruorwXJxmXaP1HQZonytECNU/UOzP6GN
Ldq0eFDE4b04Wjp396551G99YiFP2nqHVJ5OMQIDAQABAoIBAQDEeYZhjyq+avUu
eSuFhOaIU4/ZhlXycsOqzpwJvzEz61tBSvrZPA5LSb9pzAvpic+7hDH94jX89+8d
NfO7qlADsVNEQJBxuv2o1MCjpCRkmBZz506IBGU60Kt1j5kwdCEergTW1q375z4w
l8f7LmSL2U6WvKcdojTVxohBkIUJ7shtmmukDi2YnMfe6T/2JuXDDL8rvIcnfr5E
MCgPQs+xLeLEGrIJdpUy1iIYZYrzvrpJwf9EJL3D0e7jkpbvAQZ8EF9YhEizJhOm
dzTqW4PgW2yUaHYd3q5QjiILy7AC+oOYoTZln3RfjPOxl+bYjeMOWlqkgtpPQkAE
4I64w8RZAoGBAPLR44pEkmTdfIIF8ZtzBiVfDZ29bT96J0CWXGVzp8x6bSu5J5jl
s7sP8DEcjGZ6vHsLGOvkcNxzcnR3l/5HOz6TIuvVuUm36b1jHltq1xZStjGeKZs1
ihhJSu2lIA+TrK8FCRnKARJ0ughXGNZFItgeM230Sgjp2RL4ISXJ724XAoGBANBy
S2RwNpUYvkCSZHSFnQM/jq1jldxw+0p4jAGpWLilEaA/8xWUnZrnCrPFF/t9llpb
dTR/dCI8ntIMAy2dH4IUHyYKUahyHSzCAUNKpS0s433kn5hy9tGvn7jyuOJ4dk9F
o1PIZM7qfzmkdCBbX3NF2TGpzOvbYGJHHC3ssVr3AoGBANHJDopN9iDYzpJTaktA
VEYDWnM2zmUyNylw/sDT7FwYRaup2xEZG2/5NC5qGM8NKTww+UYMZom/4FnJXXLd
vcyxOFGCpAORtoreUMLwioWJzkkN+apT1kxnPioVKJ7smhvYAOXcBZMZcAR2o0m0
D4eiiBJuJWyQBPCDmbfZQFffAoGBAKpcr4ewOrwS0/O8cgPV7CTqfjbyDFp1sLwF
2A/Hk66dotFBUvBRXZpruJCCxn4R/59r3lgAzy7oMrnjfXl7UHQk8+xIRMMSOQwK
p7OSv3szk96hy1pyo41vJ3CmWDsoTzGs7bcdMl72wvKemRaU92ckMEZpzAT8cEMC
cWKLb8yzAoGAMibG8IyHSo7CJz82+7UHm98jNOlg6s73CEjp0W/+FL45Ka7MF/lp
xtR3eSmxltvwvjQoti3V4Qboqtc2IPCt+EtapTM7Wo41wlLCWCNx4u25pZPH/c8g
1yQ+OvH+xOYG+SeO98Phw/8d3IRfR83aqisQHv5upo2Rozzo0Kh3OsE=
-----END RSA PRIVATE KEY-----`,
	ClientKeys: map[uint32]string{
		2: `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAz/fyfozlDIDWG9e3Lb29+7j3c66wvUJBaBWP10rB9HTE6prj
fcGMqC9imr6zAdD9q+Gr1j7egvqgi3Da+VBAMFH92/5wD5PsD7dX8Z2f4o65Vk2n
VOY8Dl75Z/uRhg0Euwnfrved69z9LG6utmlyv6YUPAflXh/JFw7Dq6c4EGeR+Kej
FTwmVhEdzPGHjXhFmsVt9HdXRYSf4NxHPzOwj8tiSaOQA0jC4E4mM7rvGSH5GX6h
ma+7pJnl/5+rEVM0mSQvm0m1XefmuFy040bEZ/6O7ZenOGBsvvwuG3TT4FNDNzW8
Dw9ExH1l6NoRGaVkDdtrl/nFu5+a09Pm/E0ElwIDAQABAoIBAQCtH17Cck+KJQYX
j29xqG4qykNUDawbILiKCMkBE7553Wq/UcjmuuR4bVnML8ucS3mgR/BgHV3l8vUK
nxvqRx/oGZkWNazbiuwL+ThAblLWqrEmYuZVCoQcAnvkT8tIqDWz7fhDEuZnnkMz
ZcATIZzgZUSa5IfP3u3rP+MrVbyaCdzJEeI0Yrv1XT+M5ddkKQrYgqC5kRiYi/Lj
NcLJhqSVt8p37CdJx1PGHFjKKb4MZpANlNRgeTtWpGVfS0PJLzaiI1NyPSJv7xWZ
gVhbK9+wQxqSG6KmZ4vpEvRI1zKiov5BsAFN+GfuD5mpn1Xo9CpzTfj/sO13VpHH
+Mt80+yBAoGBAPYXVEcXug5zqkqXup4dp1S05saz1zWPhUhQm+CrbhgeTqpjngJJ
EB79qMrGmyki0P/cGtbTcrHf8+i7gDlIGW0OMb4/jn4f5ACVD00iyvkHSGPn0Aim
MoNOMbkGot7SkSnncwxXdawwDyTu2dofXuBr72+GYqgRAG52IuA0C0pRAoGBANhX
p/UyW/htB27frKch/rTKQKm12kBV20AkkRUQUibiiQyWueWKs+5bVaW5R5oDIhWx
qftJtnEFWUvWaTHpHsB/bpjS3CJ6WknqNbpa3QIScpV1uw8V+Etz/K2/ftjyZzFo
nqc+Jud5364xFdIlOsRj9gZnK83Wcui6EFxAer5nAoGBAJzTzzSjLUHqejqhKR98
nFeCFZPJpjuO5AxqunvaJAYgwlcZtueT8j8dvgTDvrvfYTu85CnFhNFQfFrzqspW
ZUW3hwHL9R3xatboJ2Er7Bf5iSuJ3my0pXpCSbO1Q/QmUrZWtl3GGsqJsg0CXjkA
RvFUN7ll9ddPRmwewykIYa2RAoGAcmKuWFNfE1O6WWIELH4p6LcDR3fyRI/gk+KB
nyx48zxVkAVllrsmdYFvIGd9Ny4u6F9+a3HG960HULS1/ACxFMCL3lumrsgYUvp1
m+mM7xqH4QRVeh14oZRa5hbY36YS76nMMMsI0Ny8aqJjUjADCXF81FfabkPTj79J
BS3GeEMCgYAXmFIT079QokHjJrWz/UaoEUbrNkXB/8vKiA4ZGSzMtl3jUPQdXrVf
e0ofeKiqCQs4f4S0dYEjCv7/OIijV5L24mj/Z1Q4q++S5OksKLPPAd3gX4AYbRcg
PS4rUKl1oDk/eYN0CNYC/DYV9sAv65lX8b35HYhvXISVYtwwQu/+Yg==
-----END RSA PRIVATE KEY-----`,
		3: `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA02M1I1V/YvxANOvLFX8R7D8At40IlT7HDWpAW3t+tAgQ7sqj
CeYOxiXqOaaw2kJhM3HT5nZll48UmykVq45Q05J57nhdSsGXLJshtLcTg9liMEoW
61BjVZi9EPPRSnE05tBJc57iqZw+aEcaSU0awfzBc8IkRd6+pJ5iIgEVfuTluani
zhHWvRli3EkAF4VNhaTfP3EkYfr4NE899aUeScbbdLFI6u1XQudlJCPTxaISx5Zc
wM+nP3v242ABcjgUcfCbz0AY547WazK4bWP3qicyxo4MoLOoe9WBq6EuG4CuZQrz
Knq8ltSxud/6chdg8Mqp/IasEQ2TpvY78tEXDQIDAQABAoIBAQC4uPsYk4AsSe75
0Au6Dz7kSfIgdDhJ44AisvTmfLauMFZLtfxfjBDhCwTxuD7XnCZAxHm97Ty+AqSp
Km/raQQsvtWalMhBqYanzjDYMRv2niJ1vGjm3WrQxBaEF+yOtvrZsK5fQTslqInI
qknIQH7fgjazJ7Z28D18sYNj37qfFWSSymgFo+SoS/BKEr200lpRA/oaGXiHcyIO
jJidP6b7UGes7uhMXUvLrfozmCsSqslxXO5Uk5XN/fWl4LxCGX7mpNfPZIT5YBSj
HliFkNlxIjyJg8ORLGi82M2cuyxp39r93F6uaCjLtb+rdwlGur7npgXUkKfWQJf9
WE7uar6BAoGBAPXIuIuYFFUhqNz5CKU014jZu6Ql0z5ZA08V84cTJcfLIK4e2rqC
8DFTldA0FtVfOGt0V08H/x2pRChGOvUwGG5nn9Dqqh6BjByUrW4z2hnXzT3ZuSDh
6eapiCB1jl9meJ0snhF2Ps/hqWGL2b3SkCCe90qVTzOVOeLO6YUCIOq9AoGBANws
fQkAq/0xw8neRGNTrnXimvbS+VXPIF38widljubNN7DY5cIFTQJrnTBWKbuz/t9a
J8QX6TFL0ci/9vhPJoThfL12vL2kWGYgWkWRPmqaBW3yz7Hs5rt+xuH3/7A5w5vm
kEg1NZJgnsJ0rMUTu1Q6PM5CBg6OpyHY4ThBb8qRAoGAML8ciuMgtTm1yg3CPzHZ
xZSZeJbf7K+uzlKmOBX+GkAZPS91ZiRuCvpu7hpGpQ77m6Q5ZL1LRdC6adpz+wkM
72ix87d3AhHjfg+mzgKOsS1x0WCLLRBhWZQqIXXvRNCH/3RH7WKsVoKFG4mnJ9TJ
LQ8aMLqoOKzSDD/JZM3lRWkCgYA8hn5Y2zZshCGufMuQApETFxhCgfzI+geLztAQ
xHpkOEX296kxjQN+htbPUuBmGTUXcVE9NtWEF7Oz3BGocRnFrbb83odEGsmySXKH
bUYbR/v2Ham638UOBevmcqZ3a2m6kcdYEkiH1MfP7QMRqjr1DI1qpfvERLLtOxGu
xU5WAQKBgQCaVavyY6Slj3ZRQ7iKk9fHkge/bFl+zhANxRfWVOYMC8mD2gHDsq9C
IdCp1Mg0tJpWLaGgyDM1kgChZYsff4jRxHC4czvAtoPSlxWXF2nL31qJ3vk2Zzzc
a4GSHAInodXBrKstav5SIKosWKT2YysxgHlA9Sm2f4n09GjFbslEHg==
-----END RSA PRIVATE KEY-----`,
		4: `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAyaxqjPJP5+Innfv5IdfQqY/ftS++lnDRe3EczNkIjESWXhHS
OljEw9b9C+/BtF+fO9QZL7Z742y06eIdvsMPQKdGflB26+9OZ8AF4SpXDn3aVWGr
8+9qpB7BELRZI/Ph2FlFL4cobCzMHunncW8zTfMId48+fgHkAzCjRl5rC6XT0Yge
6+eKpXmF+hr0vGYWiTzqPzTABl44WZo3rw0yurZTzkrmRE4kR2VzkjY/rBnQAbFK
KFUKsUozjCXvSag4l461wDkhmmyivpNkK5cAxuDbsmC39iqagMt9438fajLVvYOv
pVs9ci5tiLcbBtfB4Rf/QVAkqtTm86Z0O3e7DwIDAQABAoIBAQCyma226vTW35LE
N5zXWuAg+hhcxk6bvofWMUMXKvGF/0vHPTMXlvuSkDeDNa4vBivneRthBNPMgb3q
DuTWxrogQMOOI8ZdhY3DFexfDvcQD2anDJuSqSmg9Nd36q+yxk3xIoXB5Ilo23dd
vTnJXHhsBNovv7zRLO134cAHFqDoKzt5EEHre0skUcn6HjHOek6c53jvpKr5LSrr
iwx5gMuY/7ZSIUDo9WGY70qbQFGY6bOlX9x8uNjcFF+7SztEVQ+vhJ/+7EvwqaJr
ysweo0l91TKM9WaMuwoucKeceVWuynEw6GGTw8UTLtltekLGe6bS8YxY8fVwnKkT
RwJYwAJRAoGBAP2rhcfOA+1Ja37hUHKebfp9rHsex4+pGyt3Kdu7WdqOn4sexmya
BuiHQcUchPDVla/ruQZ20+8LHgzBDo0m8sY7gpf715UV9NSVIRD0wu26SKRklOFz
J4HBOwU9hBGLSnRUJzyvVlt5O7E9hAv61SCrvWBEcow2YnKNQLwvjMVJAoGBAMuG
oSb3A/ulqtp2zpxVAclYe/bSItZZTOUWP6Vb4hOiHxIJ0n1H9ap6grOYkJ/Yn4gg
yYzKm/noF1wXP7Rj/xOahnvMkzhGdmOabvE9LH5HwQTWxBBWTkZzgBbYtbg+J5MT
cKqJaychSRjJj+xX+d90rtlSu/c27chlSRKAHXWXAoGAFTcIHDq9l1XBmL3tRXi8
h+uExlM/q2MgM5VmucrEbAPrke4D+Ec1drMBLCQDdkTWnPzg34qGlQJgA/8NYX61
ZSDK/j0AvaY1cKX8OvfNaaZftuf2j5ha4H4xmnGXnwQAORRkp62eUk4kUOFtLrdO
pcnXL7rpvZI6z4vCszpi0okCgYEAp3lZEl8g/+oK9UneKfYpSi1tlGTGFevVwozU
QpWhKta1CnraogycsnOtKWvZVi9C1xljwF7YioPY9QaMfTvroY3+K9DjM+OHd96U
fB4Chsc0pW60V10te/t+403f+oPqvLO6ehop+kEBjUwPCkQ6cQ3q8xmJYpvofoYZ
4wdZNnECgYBwG8Vrv7Z+kX9Zuh1FvcRoY57bYLU0cWW92SA3Nvi8pZOIEaLHrQyZ
pvvaLIicR1m9+KsOAmii7ru0zL7KsrGW+5migQsaDi4gzahKQpad/R7MLKi/L53r
Ymo0aZKARLHW82GbomQ0zxdRoo9vaqfGNpXkxyyt3k3GGDunmrskYw==
-----END RSA PRIVATE KEY-----`,
		5: `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAsJbFp3WcsiojjdQtVnTuvtawL2m4XxK93F6lCnFwcZqUP39t
xFGGlrogHMqreyawIUN7E5shtwGzigzjW8Ly5CryBJpXP3ehNTqJS7emb+9LlC19
Oxa1eQuUQnatgcsd16DPH7kJ5JzN3vXnhvUyk4Qficdmm0uk7FRaNYFi7EJs4xyq
FTrp3rDZ0dzBHumlIeK1om7FNt6Nyivgp+UybO7kl0NLFEeSlV4S+7ofitWQsO5x
YqKAzSzz+KIRQcxJidGBlZ1JN/g5DPDpx/ztvOWYUlM7TYk6xN3focZpU0kBzAw/
rn94yW9z8jpXfzk+MvWzVL/HAcPy4ySwkay0NwIDAQABAoIBADzKWpawbVYEHaM4
lLb7oCjAPXzE9zx7djLDvisfLCdfoINPedkoe52ty1o+BtRpWB7LXTY9pFic1FLE
5wvyy6zyf8hH3ZsysqNhWFxhh4FnLmx/UGokAir+anaK5mYVJ1vQtxzjlV1HAbQs
kRyrklKoHDdRFqiFXOwiib97oDNWhD+RxfyGwwJnynZZSXdLbLSiz/QHQNr/+Ufk
KRBaxt0CfU7mOLZxoy6fNAxHdBcBJPHCyh+aDvEbix7nSncSU8Ju/48YJ8DrglbZ
sXCYoA5Uz8NMDuaEMgoNWCFQVoEcRkEUoaH7BlWd3UUFRPnDZ1B4BmkrVoRE8a58
3OqSwakCgYEA19wQUISXtpnmCrEZfbyZ6IwOy8ZCVaVUtbTjVa8UyfNglzzJG3yz
cXU3X35v5/HNCHaXbG2qcbQLThnHBA+obW3RDo+Q49V84Zh1fUNH0ONHHuC09kB/
/gHqzn/4nLf1aJ2O0NrMyrZNsZ0ZKUKQuVCqWjBOmTNUitcc8RpXZ8sCgYEA0W09
POM/It7RoVGI+cfbbgSRmzFo9kzSp5lP7iZ81bnvUMabu2nv3OeGc3Pmdh1ZJFRw
6iDM6VVbG0uz8g+f8+JT32XdqM7MJAmgfcYfTVBMiVnh330WNkeRrGWqQzB2f2Wr
+0vJjU8CAAcOWDh0oNguJ1l1TSyKxqdL8FsA38UCgYEAudt1AJ7psgOYmqQZ+rUl
H6FYLAQsoWmVIk75XpE9KRUwmYdw8QXRy2LNpp9K4z7C9wKFJorWMsh+42Q2gzyo
HHBtjEf4zPLIb8XBg3UmpKjMV73Kkiy/B4nHDr4I5YdO+iCPEy0RH4kQJFnLjEcQ
LT9TLgxh4G7d4B2PgdjYYTkCgYEArdgiV2LETCvulBzcuYufqOn9/He9i4cl7p4j
bathQQFBmSnkqGQ+Cn/eagQxsKaYEsJNoOxtbNu/7x6eVzeFLawYt38Vy0UuzFN5
eC54WXNotTN5fk2VnKU4VYVnGrMmCobZhpbYzoZhQKiazby/g60wUtW9u7xXzqOd
M/428YkCgYBwbEOx1RboH8H+fP1CAiF+cqtq4Jrz9IRWPOgcDpt2Usk1rDweWrZx
bTRlwIaVc5csIEE2X02fut/TTXr1MoXHa6s2cQrnZYq44488NsO4TAC26hqs/x/H
bVOcX13gT26SYngAHHeh7xjWJr/KgIIwvcvgvoVs6lu7a8aLUvrOag==
-----END RSA PRIVATE KEY-----`,
	},
}
