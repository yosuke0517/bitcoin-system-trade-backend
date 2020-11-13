
-- +migrate Up
CREATE TABLE IF NOT EXISTS `SIGNAL_EVENTS` (
  `time` TIMESTAMP PRIMARY KEY NOT NULL DEFAULT '2020-01-01 00:00:01',
  `product_code` VARCHAR(50),
  `side` VARCHAR(50),
  `price` float,
  `size` float
);
-- +migrate Down
DROP TABLE IF EXISTS `SIGNAL_EVENTS`;