package config

var (
	WELCOME_MESSAGE string
	WORKERS_NUM     int
	TMP_FILE_PREFIX string
)

func LoadEnv() {
	TMP_FILE_PREFIX = "tmpfile"
	WELCOME_MESSAGE = "asd"
	WORKERS_NUM = 5
}
