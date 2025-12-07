CREATE TABLE IF NOT EXISTS cells
(
    id                 SERIAL PRIMARY KEY,

    -- общий тип и версия технологий
    tech               TEXT NOT NULL,

    -- общие поля
    mcc                INT  NOT NULL, --код страны
    mnc                INT  NOT NULL, --код сети мобильной связи

    -- GSM, WCDMA: LAC
    lac                INT,           --код зоны местоположения

    --LTE: TAC
    tac                INT,           --код зоны отслеживания

    --GSM/WCDMA: CID
    cid                INT,           --уникальный идентификатор соты

    --LTE: CI
    ci                 INT,           --уникальный идентификатор соты

    signal_strength    INT,           --текущая мощность сигнала, dBm

    --GSM
    bsic               INT,           --код базовой станции
    arfcn              INT,           --абсолютный радиочастотный номер канала
    gsm_timing_advance INT,           --значение опережения синхронизации
    gsm_age            INT,           --как давно получен сигнал

    --WCDMA
    psc                INT,           --первичный скремблирующий код
    uarfcn             INT,           --абсолютный радиочастотный номер канала
    wcdma_age          INT,           --кол-во мс с тех пор, как эта сота была основной

    --LTE
    pci                INT,           --физический идентификатор соты
    earfcn             INT,           --абсолютный радиочастотный номер канала
    lte_timing_advance INT,           --значение опережения синхронизации
    lte_age            INT,           --кол-ва мс с тех пор, как эта сота была основной


    --coordinates
    lat                DOUBLE PRECISION,
    lon                DOUBLE PRECISION,

    from_metro         BOOLEAN DEFAULT FALSE
);

CREATE UNIQUE INDEX IF NOT EXISTS cells_idx_unique
    ON cells (tech, mcc, mnc, COALESCE(lac, tac), COALESCE(cid, ci));

CREATE TABLE IF NOT EXISTS wifi
(
    id              SERIAL PRIMARY KEY,

    bssid           VARCHAR(17), --MAC-адрес узла
    signal_strength INT,         --текущая мощность сигнала, dBm
    channel         INT,         --канал взаимодействия с точкой доступа
    age             INT          --как давно получен сигнал, мс
);

CREATE TABLE IF NOT EXISTS ip
(
    id      SERIAL PRIMARY KEY,
    address VARCHAR(17) --IPv4 или IPv6 адрес устройства
);