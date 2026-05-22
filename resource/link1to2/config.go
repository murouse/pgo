package link1to2

// Config содержит параметры конфигурации для generic-ресурса,
// управляющего связями «многие-ко-многим» (Many-to-Many) между двумя таблицами.
type Config struct {
	LinkTable string // имя связующей (cross-reference) таблицы (например, "multirack_tags").

	LeftColumnID string // имя колонки левой сущности в связующей таблице (например, "multirack_id").
	LeftTable    string // имя таблицы левой (родительской) сущности (например, "multiracks").

	RightColumnID string // имя колонки правой сущности в связующей таблице (например, "tag_id").
	RightTable    string // имя таблицы правой сущности (например, "tags").

	DataIntegrityErr error //  ошибка, возвращаемая в случае нарушения целостности данных.
}
