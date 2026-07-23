-- +goose Up
-- Состав групп материалов M: какие материалы «накрывает» формула, заданная на группу.
-- Один материал может входить в несколько групп.
CREATE TABLE material_groups (
    group_m       text    NOT NULL,
    material_id   bigint  NOT NULL,
    material_name text    NOT NULL,
    name_lvl_2    text,
    name_lvl_3    text,
    tlevel        text,
    valid_from    date    NOT NULL DEFAULT '1900-01-01',
    valid_to      date    NOT NULL DEFAULT '9999-12-31',
    is_leaf       boolean NOT NULL DEFAULT true,

    PRIMARY KEY (group_m, material_id)
);

-- Подбор формулы идёт от материала строки спроса к его группам.
CREATE INDEX material_groups_material_id_idx ON material_groups (material_id);

COMMENT ON TABLE material_groups IS 'Справочник состава групп материалов M (источник: material_groups.csv, колонки hname/code_nsi)';
COMMENT ON COLUMN material_groups.group_m IS 'Код группы M (hname), например MT00000116';
COMMENT ON COLUMN material_groups.material_id IS 'Код материала НСИ (code_nsi) — соответствует ssp.mtr_nsi_code';
COMMENT ON COLUMN material_groups.valid_from IS 'Действует с (datuv)';
COMMENT ON COLUMN material_groups.valid_to IS 'Действует по (datub)';

-- +goose Down
DROP TABLE material_groups;
