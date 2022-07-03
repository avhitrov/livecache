# livecache

Специфический кешатор, разработанный для удовлетворения задачи минимизации задержек на отдачу контента в ущерб поддержания актуальности данных. Более простыми словами - контент с истекшим сроком годности будет отдаваться на запросы вплоть до момента его фактического обновления из источника.

## Применимость

Вкратце - кеширование данных, которые имеют тяжеловесную или неустойчивую функцию обновления, и которые необходимо быстро отдавать, даже если они не очень актуальны. Например:
1. Любые in-memory кеши словарей или других данных, фактическое обновление которых происходит нечасто, а востребованность - наоборот - высокая.
2. Хранение однотипных данных (например, профили пользователей, посчитанные векторы или ab-данные) в ограниченных по размеру кластерах (бакетах).

## Структура

Кешатор состоит из двух основных единиц:

### Объект CacheItem (элемент кеша)

Самостоятельная самообновляющаяся единица кеша, в которой сложены: 
- анонимный интерфейс для хранения произвольных данных
- ttl, определяющий фиксированное время жизни кеша
- передаваемая при инициализации функция getter, возвращающая обновленное значение или ошибку. На вход принимает контекст от внешней среды.

Существенная задержка на выдачу возникает только при первом обращении к элементу кеша, когда интерфейс пуст и вынужден ждать первичного срабатывания getter. В дальнейшем данные выдаются всегда из кеша; при достижении ttl запускается горутина, выполняющая getter и перезаписывающая значение кеша после получения успешного результата. В случае получения ошибки, кеш приводится к предыдущему состоянию и запускает горутину обновления при следующей попытке чтения. Одновременный запуск более одной горутины обновления содержимого исключен.

CacheItem может быть использован самостоятельно для хранения, например, словаря или же в составе CacheBucket.

#### Пример
Инициализация:
 
    func NewFactory(ctx context.Context) *ServiceFactory {
        coldStartCache := livecache.NewCacheItem(
            func(ctx context.Context) (interface{}, error) {
                return someservice.FunctionToRetrieveData(ctx, someParamFromAbove, anotherParamFromAbove)
            },
            DefaultCacheTTLDurationType,
            DefaultGetterTTLDurationType,
        )
    
        // При необходимости можно прогреть:
        _, err := coldStartCache.Get(ctx)
        if err != nil {
            fmt.Errorf("%v - cache heating error", err)
        }
    
        return &ServiceFactory{
            serviceCache: coldStartCache,
        }
    }

Использование:

    func (s *Service) DataRetriever(ctx context.Context) ([]string, error) {
        cacheItem, err := s.recommendationsRecmicColdCache.Get(ctx)
        if err != nil {
            return nil, err
        }
    
        cacheResult, ok := cacheItem.([]string)
        if !ok {
            return nil, errors.New("unexpected data type for cacheResult")
        }
    
        return cacheResult, nil
    }

### Объект CacheBucket

Мапа из объектов CacheItem, которая может быть ограничена по размеру или времени жизни элементов. В случае наличия любого из ограничителей, при инициализации запускается сборщик мусора, который раз в 100 мс (умолчание, может быть переписано) проверяет размер и состояние бакета на соответствие ограничениям. 
- В случае выхода последнего времени доступа элементов кеша (LastAccessed) за пределы clearInterval, указанные элементы удаляются. 
- В случае выхода размера за ограничение, удаляются наиболее старые по LastAccessed значения. Сложность алгоритма определения "выселенцев" - O(N*ln(n))

TTL всех элементов бакета одинаковый, задается при инициализации бакета. Создание элемента кеша происходит при первом обращении к методу Get бакета, при этом геттер сохраняется в CacheItem и в дальнейшем не обновляется.

#### Пример
Инициализация:

    type Rating struct {
        ...
        similarsPackageCache livecache.CacheBucket
    }
    
    func (r *Rating) Initialize() {
        r.similarsPackageCache = livecache.NewCacheBucket(
            CacheTTLDurationType,  // Время истечения данных во всех элементах
            GetterTTLDurationType, // Максимальное время работы геттера
            nil,                   // Ключи не удаляются
            0,                     // Количество ключей не ограничено
        )
    }

Использование:

    func (r *Rating) DataRetriever(ctx context.Context, packageID string) ([]schema.ResponseItem, error) {
        getter := func(ctx context.Context) (interface{}, error) {
            similars, err := r.FunctionToRetrieveData(ctx, packageID)
            return similars, err
        }
    
        res, err := r.similarsPackageCache.Get(ctx, packageID, getter)
        if r.similarsPackageCache.IsDataEmpty(err) {
            return nil, nil
        }
        if err != nil {
            return nil, err
        }
        result, ok := res.([]*models.Gravity)
        if !ok {
            return nil, errors.New("can't cast interface{} to []*models.Gravity")
        }
    }
