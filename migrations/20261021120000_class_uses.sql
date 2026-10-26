-- +goose Up
-- AOC-081: what each class USES, which is what the Armory's class filter shows for it besides the
-- items restricted to it. Pierre, 2026-10-14 (DECISIONS.md; his table is in AOC-081's Log): "it
-- should show all the restricted to DT items but also all the items that my class should wear".
-- Armour: "we will only show the highest except for soldier we show heavy and FP options".
-- Weapons: everything the class can use. Talismans: "tali for necro, demo, dt". "ring neck back
-- for all classes". Ammunition goes to every class too: each one uses a bow or a crossbow, and the
-- data's one `ammunition` type cannot tell arrows from bolts. Neither can the names (measured
-- 2026-10-21: 9 say arrow, 14 bolt, 20 neither).
--
-- ⚠️ "Uses" is not "can wear". classes.max_armour_weight, the gear builder's ceiling, stays NULL: a
-- Dark Templar uses full plate and heavy but may wear anything (DECISIONS.md 2026-09-12 and
-- 2026-10-21).
-- Rows, never slugs in code (reference/content-model.md § 0). Each pair is listed once, by slug, so
-- a mistyped slug is a NULL that NOT NULL refuses. 15 weight pairs, 115 type pairs.
CREATE TABLE class_armour_weights (
    class_id integer NOT NULL REFERENCES classes(id),
    armour_weight_id integer NOT NULL REFERENCES armour_weights(id),
    PRIMARY KEY (class_id, armour_weight_id)
);
CREATE TABLE class_item_types (
    class_id integer NOT NULL REFERENCES classes(id),
    item_type_id integer NOT NULL REFERENCES item_types(id),
    PRIMARY KEY (class_id, item_type_id)
);

INSERT INTO class_armour_weights (class_id, armour_weight_id) VALUES
    ((SELECT id FROM classes WHERE slug = 'assassin'), (SELECT id FROM armour_weights WHERE slug = 'cloth')),
    ((SELECT id FROM classes WHERE slug = 'barbarian'), (SELECT id FROM armour_weights WHERE slug = 'light')),
    ((SELECT id FROM classes WHERE slug = 'bear-shaman'), (SELECT id FROM armour_weights WHERE slug = 'medium')),
    ((SELECT id FROM classes WHERE slug = 'conqueror'), (SELECT id FROM armour_weights WHERE slug = 'full-plate')),
    ((SELECT id FROM classes WHERE slug = 'conqueror'), (SELECT id FROM armour_weights WHERE slug = 'heavy')),
    ((SELECT id FROM classes WHERE slug = 'dark-templar'), (SELECT id FROM armour_weights WHERE slug = 'full-plate')),
    ((SELECT id FROM classes WHERE slug = 'dark-templar'), (SELECT id FROM armour_weights WHERE slug = 'heavy')),
    ((SELECT id FROM classes WHERE slug = 'demonologist'), (SELECT id FROM armour_weights WHERE slug = 'cloth')),
    ((SELECT id FROM classes WHERE slug = 'guardian'), (SELECT id FROM armour_weights WHERE slug = 'full-plate')),
    ((SELECT id FROM classes WHERE slug = 'guardian'), (SELECT id FROM armour_weights WHERE slug = 'heavy')),
    ((SELECT id FROM classes WHERE slug = 'herald-of-xotli'), (SELECT id FROM armour_weights WHERE slug = 'cloth')),
    ((SELECT id FROM classes WHERE slug = 'necromancer'), (SELECT id FROM armour_weights WHERE slug = 'cloth')),
    ((SELECT id FROM classes WHERE slug = 'priest-of-mitra'), (SELECT id FROM armour_weights WHERE slug = 'light')),
    ((SELECT id FROM classes WHERE slug = 'ranger'), (SELECT id FROM armour_weights WHERE slug = 'medium')),
    ((SELECT id FROM classes WHERE slug = 'tempest-of-set'), (SELECT id FROM armour_weights WHERE slug = 'light'))
ON CONFLICT DO NOTHING;

INSERT INTO class_item_types (class_id, item_type_id) VALUES
    ((SELECT id FROM classes WHERE slug = 'assassin'), (SELECT id FROM item_types WHERE slug = 'dagger')),
    ((SELECT id FROM classes WHERE slug = 'assassin'), (SELECT id FROM item_types WHERE slug = 'crossbow')),
    ((SELECT id FROM classes WHERE slug = 'assassin'), (SELECT id FROM item_types WHERE slug = 'ring')),
    ((SELECT id FROM classes WHERE slug = 'assassin'), (SELECT id FROM item_types WHERE slug = 'necklace')),
    ((SELECT id FROM classes WHERE slug = 'assassin'), (SELECT id FROM item_types WHERE slug = 'back')),
    ((SELECT id FROM classes WHERE slug = 'assassin'), (SELECT id FROM item_types WHERE slug = 'ammunition')),
    ((SELECT id FROM classes WHERE slug = 'barbarian'), (SELECT id FROM item_types WHERE slug = '1he')),
    ((SELECT id FROM classes WHERE slug = 'barbarian'), (SELECT id FROM item_types WHERE slug = '1hb')),
    ((SELECT id FROM classes WHERE slug = 'barbarian'), (SELECT id FROM item_types WHERE slug = '2he')),
    ((SELECT id FROM classes WHERE slug = 'barbarian'), (SELECT id FROM item_types WHERE slug = '2hb')),
    ((SELECT id FROM classes WHERE slug = 'barbarian'), (SELECT id FROM item_types WHERE slug = 'bow')),
    ((SELECT id FROM classes WHERE slug = 'barbarian'), (SELECT id FROM item_types WHERE slug = 'thrown')),
    ((SELECT id FROM classes WHERE slug = 'barbarian'), (SELECT id FROM item_types WHERE slug = 'ring')),
    ((SELECT id FROM classes WHERE slug = 'barbarian'), (SELECT id FROM item_types WHERE slug = 'necklace')),
    ((SELECT id FROM classes WHERE slug = 'barbarian'), (SELECT id FROM item_types WHERE slug = 'back')),
    ((SELECT id FROM classes WHERE slug = 'barbarian'), (SELECT id FROM item_types WHERE slug = 'ammunition')),
    ((SELECT id FROM classes WHERE slug = 'bear-shaman'), (SELECT id FROM item_types WHERE slug = '2hb')),
    ((SELECT id FROM classes WHERE slug = 'bear-shaman'), (SELECT id FROM item_types WHERE slug = 'bow')),
    ((SELECT id FROM classes WHERE slug = 'bear-shaman'), (SELECT id FROM item_types WHERE slug = 'crossbow')),
    ((SELECT id FROM classes WHERE slug = 'bear-shaman'), (SELECT id FROM item_types WHERE slug = 'thrown')),
    ((SELECT id FROM classes WHERE slug = 'bear-shaman'), (SELECT id FROM item_types WHERE slug = 'ring')),
    ((SELECT id FROM classes WHERE slug = 'bear-shaman'), (SELECT id FROM item_types WHERE slug = 'necklace')),
    ((SELECT id FROM classes WHERE slug = 'bear-shaman'), (SELECT id FROM item_types WHERE slug = 'back')),
    ((SELECT id FROM classes WHERE slug = 'bear-shaman'), (SELECT id FROM item_types WHERE slug = 'ammunition')),
    ((SELECT id FROM classes WHERE slug = 'conqueror'), (SELECT id FROM item_types WHERE slug = '1he')),
    ((SELECT id FROM classes WHERE slug = 'conqueror'), (SELECT id FROM item_types WHERE slug = '1hb')),
    ((SELECT id FROM classes WHERE slug = 'conqueror'), (SELECT id FROM item_types WHERE slug = '2he')),
    ((SELECT id FROM classes WHERE slug = 'conqueror'), (SELECT id FROM item_types WHERE slug = '2hb')),
    ((SELECT id FROM classes WHERE slug = 'conqueror'), (SELECT id FROM item_types WHERE slug = 'crossbow')),
    ((SELECT id FROM classes WHERE slug = 'conqueror'), (SELECT id FROM item_types WHERE slug = 'thrown')),
    ((SELECT id FROM classes WHERE slug = 'conqueror'), (SELECT id FROM item_types WHERE slug = 'ring')),
    ((SELECT id FROM classes WHERE slug = 'conqueror'), (SELECT id FROM item_types WHERE slug = 'necklace')),
    ((SELECT id FROM classes WHERE slug = 'conqueror'), (SELECT id FROM item_types WHERE slug = 'back')),
    ((SELECT id FROM classes WHERE slug = 'conqueror'), (SELECT id FROM item_types WHERE slug = 'ammunition')),
    ((SELECT id FROM classes WHERE slug = 'dark-templar'), (SELECT id FROM item_types WHERE slug = '1he')),
    ((SELECT id FROM classes WHERE slug = 'dark-templar'), (SELECT id FROM item_types WHERE slug = '1hb')),
    ((SELECT id FROM classes WHERE slug = 'dark-templar'), (SELECT id FROM item_types WHERE slug = 'crossbow')),
    ((SELECT id FROM classes WHERE slug = 'dark-templar'), (SELECT id FROM item_types WHERE slug = 'thrown')),
    ((SELECT id FROM classes WHERE slug = 'dark-templar'), (SELECT id FROM item_types WHERE slug = 'shield')),
    ((SELECT id FROM classes WHERE slug = 'dark-templar'), (SELECT id FROM item_types WHERE slug = 'talisman')),
    ((SELECT id FROM classes WHERE slug = 'dark-templar'), (SELECT id FROM item_types WHERE slug = 'ring')),
    ((SELECT id FROM classes WHERE slug = 'dark-templar'), (SELECT id FROM item_types WHERE slug = 'necklace')),
    ((SELECT id FROM classes WHERE slug = 'dark-templar'), (SELECT id FROM item_types WHERE slug = 'back')),
    ((SELECT id FROM classes WHERE slug = 'dark-templar'), (SELECT id FROM item_types WHERE slug = 'ammunition')),
    ((SELECT id FROM classes WHERE slug = 'demonologist'), (SELECT id FROM item_types WHERE slug = 'staff')),
    ((SELECT id FROM classes WHERE slug = 'demonologist'), (SELECT id FROM item_types WHERE slug = 'dagger')),
    ((SELECT id FROM classes WHERE slug = 'demonologist'), (SELECT id FROM item_types WHERE slug = 'crossbow')),
    ((SELECT id FROM classes WHERE slug = 'demonologist'), (SELECT id FROM item_types WHERE slug = 'thrown')),
    ((SELECT id FROM classes WHERE slug = 'demonologist'), (SELECT id FROM item_types WHERE slug = 'talisman')),
    ((SELECT id FROM classes WHERE slug = 'demonologist'), (SELECT id FROM item_types WHERE slug = 'ring')),
    ((SELECT id FROM classes WHERE slug = 'demonologist'), (SELECT id FROM item_types WHERE slug = 'necklace')),
    ((SELECT id FROM classes WHERE slug = 'demonologist'), (SELECT id FROM item_types WHERE slug = 'back')),
    ((SELECT id FROM classes WHERE slug = 'demonologist'), (SELECT id FROM item_types WHERE slug = 'ammunition')),
    ((SELECT id FROM classes WHERE slug = 'guardian'), (SELECT id FROM item_types WHERE slug = '1he')),
    ((SELECT id FROM classes WHERE slug = 'guardian'), (SELECT id FROM item_types WHERE slug = '1hb')),
    ((SELECT id FROM classes WHERE slug = 'guardian'), (SELECT id FROM item_types WHERE slug = 'polearm')),
    ((SELECT id FROM classes WHERE slug = 'guardian'), (SELECT id FROM item_types WHERE slug = 'crossbow')),
    ((SELECT id FROM classes WHERE slug = 'guardian'), (SELECT id FROM item_types WHERE slug = 'thrown')),
    ((SELECT id FROM classes WHERE slug = 'guardian'), (SELECT id FROM item_types WHERE slug = 'shield')),
    ((SELECT id FROM classes WHERE slug = 'guardian'), (SELECT id FROM item_types WHERE slug = 'ring')),
    ((SELECT id FROM classes WHERE slug = 'guardian'), (SELECT id FROM item_types WHERE slug = 'necklace')),
    ((SELECT id FROM classes WHERE slug = 'guardian'), (SELECT id FROM item_types WHERE slug = 'back')),
    ((SELECT id FROM classes WHERE slug = 'guardian'), (SELECT id FROM item_types WHERE slug = 'ammunition')),
    ((SELECT id FROM classes WHERE slug = 'herald-of-xotli'), (SELECT id FROM item_types WHERE slug = '2he')),
    ((SELECT id FROM classes WHERE slug = 'herald-of-xotli'), (SELECT id FROM item_types WHERE slug = 'dagger')),
    ((SELECT id FROM classes WHERE slug = 'herald-of-xotli'), (SELECT id FROM item_types WHERE slug = 'crossbow')),
    ((SELECT id FROM classes WHERE slug = 'herald-of-xotli'), (SELECT id FROM item_types WHERE slug = 'thrown')),
    ((SELECT id FROM classes WHERE slug = 'herald-of-xotli'), (SELECT id FROM item_types WHERE slug = 'ring')),
    ((SELECT id FROM classes WHERE slug = 'herald-of-xotli'), (SELECT id FROM item_types WHERE slug = 'necklace')),
    ((SELECT id FROM classes WHERE slug = 'herald-of-xotli'), (SELECT id FROM item_types WHERE slug = 'back')),
    ((SELECT id FROM classes WHERE slug = 'herald-of-xotli'), (SELECT id FROM item_types WHERE slug = 'ammunition')),
    ((SELECT id FROM classes WHERE slug = 'necromancer'), (SELECT id FROM item_types WHERE slug = 'staff')),
    ((SELECT id FROM classes WHERE slug = 'necromancer'), (SELECT id FROM item_types WHERE slug = 'dagger')),
    ((SELECT id FROM classes WHERE slug = 'necromancer'), (SELECT id FROM item_types WHERE slug = 'crossbow')),
    ((SELECT id FROM classes WHERE slug = 'necromancer'), (SELECT id FROM item_types WHERE slug = 'thrown')),
    ((SELECT id FROM classes WHERE slug = 'necromancer'), (SELECT id FROM item_types WHERE slug = 'talisman')),
    ((SELECT id FROM classes WHERE slug = 'necromancer'), (SELECT id FROM item_types WHERE slug = 'ring')),
    ((SELECT id FROM classes WHERE slug = 'necromancer'), (SELECT id FROM item_types WHERE slug = 'necklace')),
    ((SELECT id FROM classes WHERE slug = 'necromancer'), (SELECT id FROM item_types WHERE slug = 'back')),
    ((SELECT id FROM classes WHERE slug = 'necromancer'), (SELECT id FROM item_types WHERE slug = 'ammunition')),
    ((SELECT id FROM classes WHERE slug = 'priest-of-mitra'), (SELECT id FROM item_types WHERE slug = '1he')),
    ((SELECT id FROM classes WHERE slug = 'priest-of-mitra'), (SELECT id FROM item_types WHERE slug = '1hb')),
    ((SELECT id FROM classes WHERE slug = 'priest-of-mitra'), (SELECT id FROM item_types WHERE slug = 'dagger')),
    ((SELECT id FROM classes WHERE slug = 'priest-of-mitra'), (SELECT id FROM item_types WHERE slug = 'staff')),
    ((SELECT id FROM classes WHERE slug = 'priest-of-mitra'), (SELECT id FROM item_types WHERE slug = 'crossbow')),
    ((SELECT id FROM classes WHERE slug = 'priest-of-mitra'), (SELECT id FROM item_types WHERE slug = 'thrown')),
    ((SELECT id FROM classes WHERE slug = 'priest-of-mitra'), (SELECT id FROM item_types WHERE slug = 'shield')),
    ((SELECT id FROM classes WHERE slug = 'priest-of-mitra'), (SELECT id FROM item_types WHERE slug = 'ring')),
    ((SELECT id FROM classes WHERE slug = 'priest-of-mitra'), (SELECT id FROM item_types WHERE slug = 'necklace')),
    ((SELECT id FROM classes WHERE slug = 'priest-of-mitra'), (SELECT id FROM item_types WHERE slug = 'back')),
    ((SELECT id FROM classes WHERE slug = 'priest-of-mitra'), (SELECT id FROM item_types WHERE slug = 'ammunition')),
    ((SELECT id FROM classes WHERE slug = 'ranger'), (SELECT id FROM item_types WHERE slug = '1he')),
    ((SELECT id FROM classes WHERE slug = 'ranger'), (SELECT id FROM item_types WHERE slug = '1hb')),
    ((SELECT id FROM classes WHERE slug = 'ranger'), (SELECT id FROM item_types WHERE slug = 'dagger')),
    ((SELECT id FROM classes WHERE slug = 'ranger'), (SELECT id FROM item_types WHERE slug = 'bow')),
    ((SELECT id FROM classes WHERE slug = 'ranger'), (SELECT id FROM item_types WHERE slug = 'crossbow')),
    ((SELECT id FROM classes WHERE slug = 'ranger'), (SELECT id FROM item_types WHERE slug = 'thrown')),
    ((SELECT id FROM classes WHERE slug = 'ranger'), (SELECT id FROM item_types WHERE slug = 'shield')),
    ((SELECT id FROM classes WHERE slug = 'ranger'), (SELECT id FROM item_types WHERE slug = 'ring')),
    ((SELECT id FROM classes WHERE slug = 'ranger'), (SELECT id FROM item_types WHERE slug = 'necklace')),
    ((SELECT id FROM classes WHERE slug = 'ranger'), (SELECT id FROM item_types WHERE slug = 'back')),
    ((SELECT id FROM classes WHERE slug = 'ranger'), (SELECT id FROM item_types WHERE slug = 'ammunition')),
    ((SELECT id FROM classes WHERE slug = 'tempest-of-set'), (SELECT id FROM item_types WHERE slug = '1he')),
    ((SELECT id FROM classes WHERE slug = 'tempest-of-set'), (SELECT id FROM item_types WHERE slug = '1hb')),
    ((SELECT id FROM classes WHERE slug = 'tempest-of-set'), (SELECT id FROM item_types WHERE slug = 'polearm')),
    ((SELECT id FROM classes WHERE slug = 'tempest-of-set'), (SELECT id FROM item_types WHERE slug = 'dagger')),
    ((SELECT id FROM classes WHERE slug = 'tempest-of-set'), (SELECT id FROM item_types WHERE slug = 'staff')),
    ((SELECT id FROM classes WHERE slug = 'tempest-of-set'), (SELECT id FROM item_types WHERE slug = 'bow')),
    ((SELECT id FROM classes WHERE slug = 'tempest-of-set'), (SELECT id FROM item_types WHERE slug = 'crossbow')),
    ((SELECT id FROM classes WHERE slug = 'tempest-of-set'), (SELECT id FROM item_types WHERE slug = 'thrown')),
    ((SELECT id FROM classes WHERE slug = 'tempest-of-set'), (SELECT id FROM item_types WHERE slug = 'shield')),
    ((SELECT id FROM classes WHERE slug = 'tempest-of-set'), (SELECT id FROM item_types WHERE slug = 'ring')),
    ((SELECT id FROM classes WHERE slug = 'tempest-of-set'), (SELECT id FROM item_types WHERE slug = 'necklace')),
    ((SELECT id FROM classes WHERE slug = 'tempest-of-set'), (SELECT id FROM item_types WHERE slug = 'back')),
    ((SELECT id FROM classes WHERE slug = 'tempest-of-set'), (SELECT id FROM item_types WHERE slug = 'ammunition'))
ON CONFLICT DO NOTHING;

-- Who uses an item. An item restricted to classes is used by those classes. An item restricted to
-- nobody is used by every class whose weights hold its armour weight or whose types hold its item
-- type. This is the rule's one statement: the `used_by` filter, its facet and its total read it.
CREATE VIEW item_used_by AS
SELECT ic.item_id, ic.class_id FROM item_classes ic
UNION
SELECT i.item_id, caw.class_id
FROM items i JOIN class_armour_weights caw ON caw.armour_weight_id = i.armour_weight_id
WHERE NOT EXISTS (SELECT 1 FROM item_classes ic WHERE ic.item_id = i.item_id)
UNION
SELECT i.item_id, cit.class_id
FROM items i JOIN class_item_types cit ON cit.item_type_id = i.item_type_id
WHERE NOT EXISTS (SELECT 1 FROM item_classes ic WHERE ic.item_id = i.item_id);

-- +goose Down
DROP VIEW item_used_by;
DROP TABLE class_item_types;
DROP TABLE class_armour_weights;
