
-- +migrate Up
CREATE TABLE IF NOT EXISTS `FX_BTC_JPY_1s` (
  `time` TIMESTAMP PRIMARY KEY NOT NULL DEFAULT '2020-01-01 00:00:01',
  `open` float,
  `close` float,
  `high` float,
  `low` float,
  `volume` float
);
-- +migrate Down
DROP TABLE IF EXISTS `FX_BTC_JPY_1s`;