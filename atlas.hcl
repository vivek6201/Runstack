# Atlas declarative schema and migration configuration

env "local" {
  url = "postgres://postgres:postgres@localhost:5432/coderunner?sslmode=disable"
  migration {
    dir = "file://migrations"
  }
}

env "docker" {
  url = "postgres://postgres:postgres@postgres:5432/coderunner?sslmode=disable"
  migration {
    dir = "file://migrations"
  }
}
