package dict

type Config struct {
	Table           string // название таблицы
	UniqueIndex     string // название уникального ограничения
	NotFoundErr     error  // ошибка not_found
	AlreadyExistErr error  // ошибка already_exist
}
