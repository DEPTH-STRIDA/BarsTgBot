package masker

import (
	"reflect"

	"go.uber.org/zap"
)

// LogConfigs логгирует структуры, в том числе вложенные.
// Если поле помечено тегом masked, то оно будет логгироваться замаскированным.
// Каждая структура логируется отдельной строкой. Вложенные поля не логгируются отдельно.
func LogConfigs(logger *zap.Logger, configs ...interface{}) error {
	for _, config := range configs {

		v := reflect.ValueOf(config)
		t := reflect.TypeOf(config)

		// Если config не указатель, то ошибка
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
			t = t.Elem()
		} else {
			return ErrConfigNotPointer
		}

		// Получение мапы полей
		masked := maskStructFields(v, t)

		logger.Info("Config", zap.Any(t.Name(), masked))
	}
	return nil
}

// maskStructFields маскирует поля структуры, если они отмечены тегом masked
func maskStructFields(v reflect.Value, t reflect.Type) map[string]interface{} {
	result := make(map[string]interface{})
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)
		masked := fieldType.Tag.Get("masked")

		switch field.Kind() {

		// Если поле структура, то рекурсивная обработка и добавление вложенных полей в мапу.
		case reflect.Struct:
			result[fieldType.Name] = maskStructFields(field, field.Type())

		// Если поле строка и помечено тегом masked, то маскируется.
		case reflect.String:
			if masked == "true" {
				result[fieldType.Name] = maskSensitiveData(field.String())
			} else {
				result[fieldType.Name] = field.String()
			}

		// Если поле не структура и не строка, то добавляется в мапу как есть.
		default:
			result[fieldType.Name] = field.Interface()
		}
	}
	return result
}

// maskSensitiveData маскирует строку, оставляя только первый и последний символы.
// Если строка короче 2 символов, то возвращается "****".
func maskSensitiveData(data string) string {
	if len(data) <= 2 {
		return "****"
	}
	return string(data[0]) + "****" + string(data[len(data)-1])
}
