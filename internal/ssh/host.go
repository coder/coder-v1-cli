package ssh

const coderPrefix = "coder"

func CoderHost(env string) string {
	return coderPrefix + "." + env
}
