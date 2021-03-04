package resource_test

import (
	"database/sql/driver"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/qor/admin"
	"github.com/qor/publish2"
	"github.com/qor/qor"
	"github.com/qor/qor/resource"
	testutils "github.com/qor/qor/test/utils"
	"github.com/qor/qor/utils"
	"github.com/qor/sorting"
)

func format(value interface{}) string {
	return fmt.Sprint(utils.Indirect(reflect.ValueOf(value)).Interface())
}

func checkMeta(record interface{}, meta *resource.Meta, value interface{}, t *testing.T, expectedValues ...string) {
	var (
		context       = &qor.Context{DB: testutils.TestDB()}
		metaValue     = &resource.MetaValue{Name: meta.Name, Value: value}
		expectedValue = fmt.Sprint(value)
	)

	for _, v := range expectedValues {
		expectedValue = v
	}

	meta.PreInitialize()
	meta.Initialize()

	if meta.Setter != nil {
		meta.Setter(record, metaValue, context)
		if context.HasError() {
			t.Errorf("No error should happen, but got %v", context.Errors)
		}

		result := meta.Valuer(record, context)
		if resultValuer, ok := result.(driver.Valuer); ok {
			if v, err := resultValuer.Value(); err == nil {
				result = v
			}
		}

		if format(result) != expectedValue {
			t.Errorf("Wrong value, should be %v, but got %v", expectedValue, format(result))
		}
	} else {
		t.Errorf("No setter generated for meta %v", meta.Name)
	}
}

func TestStringMetaValuerAndSetter(t *testing.T) {
	user := &struct {
		Name  string
		Name2 *string
	}{}

	res := resource.New(user)

	meta := &resource.Meta{
		Name:         "Name",
		BaseResource: res,
	}

	checkMeta(user, meta, "hello world", t)

	meta2 := &resource.Meta{
		Name:         "Name2",
		BaseResource: res,
	}

	checkMeta(user, meta2, "hello world2", t)
}

func TestIntMetaValuerAndSetter(t *testing.T) {
	user := &struct {
		Age  int
		Age2 uint
		Age3 *int8
		Age4 *uint8
	}{}

	res := resource.New(user)

	meta := &resource.Meta{
		Name:         "Age",
		BaseResource: res,
	}

	checkMeta(user, meta, 18, t)

	meta2 := &resource.Meta{
		Name:         "Age2",
		BaseResource: res,
	}

	checkMeta(user, meta2, "28", t)

	meta3 := &resource.Meta{
		Name:         "Age3",
		BaseResource: res,
	}

	checkMeta(user, meta3, 38, t)

	meta4 := &resource.Meta{
		Name:         "Age4",
		BaseResource: res,
	}

	checkMeta(user, meta4, "48", t)
}

func TestFloatMetaValuerAndSetter(t *testing.T) {
	user := &struct {
		Age  float64
		Age2 *float64
	}{}

	res := resource.New(user)

	meta := &resource.Meta{
		Name:         "Age",
		BaseResource: res,
	}

	checkMeta(user, meta, 18.5, t)

	meta2 := &resource.Meta{
		Name:         "Age2",
		BaseResource: res,
	}

	checkMeta(user, meta2, "28.5", t)
}

func TestBoolMetaValuerAndSetter(t *testing.T) {
	user := &struct {
		Actived  bool
		Actived2 *bool
	}{}

	res := resource.New(user)

	meta := &resource.Meta{
		Name:         "Actived",
		BaseResource: res,
	}

	checkMeta(user, meta, "true", t)

	meta2 := &resource.Meta{
		Name:         "Actived2",
		BaseResource: res,
	}

	checkMeta(user, meta2, "true", t)

	meta3 := &resource.Meta{
		Name:         "Actived",
		BaseResource: res,
	}

	checkMeta(user, meta3, "", t, "false")

	meta4 := &resource.Meta{
		Name:         "Actived2",
		BaseResource: res,
	}

	checkMeta(user, meta4, "f", t, "false")
}

type scanner struct {
	Body string
}

func (s *scanner) Scan(value interface{}) error {
	s.Body = fmt.Sprint(value)
	return nil
}

func (s scanner) Value() (driver.Value, error) {
	return s.Body, nil
}

func TestScannerMetaValuerAndSetter(t *testing.T) {
	user := &struct {
		Scanner scanner
	}{}

	res := resource.New(user)

	meta := &resource.Meta{
		Name:         "Scanner",
		BaseResource: res,
	}

	checkMeta(user, meta, "scanner", t)
}

func TestSliceMetaValuerAndSetter(t *testing.T) {
	t.Skip()

	user := &struct {
		Names  []string
		Names2 []*string
		Names3 *[]string
		Names4 []*string
	}{}

	res := resource.New(user)

	meta := &resource.Meta{
		Name:         "Names",
		BaseResource: res,
	}

	checkMeta(user, meta, []string{"name1", "name2"}, t)

	meta2 := &resource.Meta{
		Name:         "Names2",
		BaseResource: res,
	}

	checkMeta(user, meta2, []string{"name1", "name2"}, t)

	meta3 := &resource.Meta{
		Name:         "Names3",
		BaseResource: res,
	}

	checkMeta(user, meta3, []string{"name1", "name2"}, t)

	meta4 := &resource.Meta{
		Name:         "Names4",
		BaseResource: res,
	}

	checkMeta(user, meta4, []string{"name1", "name2"}, t)
}

type Collection struct {
	gorm.Model

	Name string

	Products       []Product `gorm:"many2many:collection_products;association_autoupdate:false"`
	ProductsSorter sorting.SortableCollection

	LunchProducts       []*Product `gorm:"many2many:collection_lunch_products;association_autoupdate:false"`
	LunchProductsSorter sorting.SortableCollection

	TagID uint
	Tag   Tag
}

type Tag struct {
	gorm.Model

	Name string
}

type Manager struct {
	gorm.Model

	Name string

	publish2.Version
}

type CollectionWithVersion struct {
	gorm.Model

	publish2.Version
	publish2.Schedule

	Name string

	Products       []ProductWithVersion `gorm:"many2many:collection_with_version_product_with_versions;association_autoupdate:false"`
	ProductsSorter sorting.SortableCollection

	ManagerID          uint
	ManagerVersionName string
	Manager            Manager
}

func (coll *CollectionWithVersion) AssignVersionName(db *gorm.DB) {
	var count int
	name := time.Now().Format("2006-01-02")
	db.Model(&CollectionWithVersion{}).Where("id = ? AND version_name like ?", coll.ID, name+"%").Count(&count)
	coll.VersionName = fmt.Sprintf("%s-v%v", name, count+1)
}

type ProductWithVersion struct {
	gorm.Model

	publish2.Schedule
	publish2.Version

	resource.CompositePrimaryKeyField

	Name string
}

type Product struct {
	gorm.Model

	Name string
}

func WithoutVersion(db *gorm.DB) *gorm.DB {
	return db.Set(admin.DisableCompositePrimaryKeyMode, "on").Set(publish2.VersionMode, publish2.VersionMultipleMode).Set(publish2.ScheduleMode, publish2.ModeOff)
}

func updateVersionPriority() func(scope *gorm.Scope) {
	return func(scope *gorm.Scope) {
		if field, ok := scope.FieldByName("VersionPriority"); ok {
			createdAtField, _ := scope.FieldByName("CreatedAt")
			createdAt := createdAtField.Field.Interface().(time.Time)

			versionNameField, _ := scope.FieldByName("VersionName")
			versionName := versionNameField.Field.Interface().(string)

			versionPriority := fmt.Sprintf("%v_%v", createdAt.UTC().Format(time.RFC3339), versionName)
			field.Set(versionPriority)
		}
	}
}
func updateCallback(scope *gorm.Scope) {
	return
}
func TestMany2ManyRelation(t *testing.T) {
	db := testutils.TestDB()
	productsMeta := setupProductsMeta(t, db, "Products")
	// lunchProduct is to cover the pointer association like []*Product
	lunchProductsMeta := setupProductsMeta(t, db, "LunchProducts")

	p1 := Product{Name: "p1"}
	p2 := Product{Name: "p2"}
	p3 := Product{Name: "p3"}
	p4 := Product{Name: "p4"}
	testutils.AssertNoErr(t, db.Save(&p1).Error)
	testutils.AssertNoErr(t, db.Save(&p2).Error)
	testutils.AssertNoErr(t, db.Save(&p3).Error)
	testutils.AssertNoErr(t, db.Save(&p4).Error)

	record := Collection{Name: "test"}
	testutils.AssertNoErr(t, db.Save(&record).Error)
	if len(record.Products) != 0 {
		t.Fatal("smoke test")
	}
	ctx := &qor.Context{DB: db}

	metaValue := &resource.MetaValue{Name: productsMeta.Name, Value: []string{fmt.Sprintf("%d", p1.ID), fmt.Sprintf("%d", p2.ID)}}
	lunchProductMetaValue := &resource.MetaValue{Name: lunchProductsMeta.Name, Value: []string{fmt.Sprintf("%d", p1.ID), fmt.Sprintf("%d", p2.ID)}}

	productsMeta.Setter(&record, metaValue, ctx)
	lunchProductsMeta.Setter(&record, lunchProductMetaValue, ctx)

	testutils.AssertNoErr(t, db.Preload("Products").Preload("LunchProducts").Find(&record).Error)
	if len(record.Products) != 2 {
		t.Error("products not set to collection")
	}
	if len(record.LunchProducts) != 2 {
		t.Error("products not set to collection")
	}
}

func setupProductsMeta(t *testing.T, db *gorm.DB, metaName string) resource.Meta {
	testutils.ResetDBTables(db, &Collection{}, &Product{})

	adm := admin.New(&qor.Config{DB: db.Set(publish2.ScheduleMode, publish2.ModeOff)})
	c := adm.AddResource(&Collection{})

	productsMeta := resource.Meta{
		Name:         metaName,
		FieldName:    metaName,
		BaseResource: c,
		Config: &admin.SelectManyConfig{
			Collection: func(value interface{}, ctx *qor.Context) (results [][]string) {
				if c, ok := value.(*Collection); ok {
					var products []Product
					ctx.GetDB().Model(c).Related(&products, "Products")

					for _, product := range products {
						results = append(results, []string{fmt.Sprintf("%v", product.ID), product.Name})
					}
				}
				return
			},
		},
	}

	var scope = &gorm.Scope{Value: c.Value}
	var getField = func(fields []*gorm.StructField, name string) *gorm.StructField {
		for _, field := range fields {
			if field.Name == name || field.DBName == name {
				return field
			}
		}
		return nil
	}

	productsMeta.FieldStruct = getField(scope.GetStructFields(), productsMeta.FieldName)

	if err := productsMeta.Initialize(); err != nil {
		t.Fatal(err)
	}

	return productsMeta
}

func TestValuer(t *testing.T) {
	db := testutils.TestDB()
	productsMeta := setupProductsMeta(t, db, "Products")

	p1 := Product{Name: "p1"}
	p2 := Product{Name: "p2"}
	testutils.AssertNoErr(t, db.Save(&p1).Error)
	testutils.AssertNoErr(t, db.Save(&p2).Error)

	record := Collection{Name: "test"}
	record.Products = []Product{p1, p2}
	testutils.AssertNoErr(t, db.Save(&record).Error)
	ctx := &qor.Context{DB: db}
	result := productsMeta.Valuer(&record, ctx)
	if len(result.([]Product)) != 2 {
		t.Error("valuer doesn't return correct value")
	}
}

func setupProductWithVersionMeta(t *testing.T, db *gorm.DB) resource.Meta {
	registerVersionNameCallback(db)
	publish2.RegisterCallbacks(db)
	testutils.ResetDBTables(db, &CollectionWithVersion{}, &ProductWithVersion{})

	adm := admin.New(&qor.Config{DB: db.Set(publish2.ScheduleMode, publish2.ModeOff)})
	c := adm.AddResource(&CollectionWithVersion{})

	productsMeta := resource.Meta{
		Name:         "Products",
		FieldName:    "Products",
		BaseResource: c,
	}

	var scope = &gorm.Scope{Value: c.Value}
	var getField = func(fields []*gorm.StructField, name string) *gorm.StructField {
		for _, field := range fields {
			if field.Name == name || field.DBName == name {
				return field
			}
		}
		return nil
	}

	productsMeta.FieldStruct = getField(scope.GetStructFields(), productsMeta.FieldName)

	if err := productsMeta.Initialize(); err != nil {
		t.Fatal(err)
	}

	return productsMeta
}

func TestValuer_WithVersion(t *testing.T) {
	db := testutils.TestDB()
	productsMeta := setupProductWithVersionMeta(t, db)

	p1 := ProductWithVersion{Name: "p1"}
	p2_v1 := ProductWithVersion{Name: "p2"}
	testutils.AssertNoErr(t, db.Save(&p1).Error)
	testutils.AssertNoErr(t, db.Save(&p2_v1).Error)
	p2_v2 := ProductWithVersion{Name: "p2"}
	p2_v2.ID = p2_v1.ID
	testutils.AssertNoErr(t, db.Save(&p2_v2).Error)

	record := CollectionWithVersion{Name: "test"}
	record.Products = []ProductWithVersion{p1, p2_v2}
	testutils.AssertNoErr(t, db.Save(&record).Error)
	ctx := &qor.Context{DB: db}

	coll := CollectionWithVersion{}
	testutils.AssertNoErr(t, db.Find(&coll).Error)
	result := productsMeta.Valuer(&coll, ctx)

	associatedProducts := result.([]ProductWithVersion)
	if len(associatedProducts) != 2 {
		t.Error("valuer doesn't return correct value")
	}

	i := 0
	for _, p := range associatedProducts {
		if p.ID == p1.ID && p.CompositePrimaryKey == fmt.Sprintf("%d%s%s", p1.ID, resource.CompositePrimaryKeySeparator, p1.GetVersionName()) {
			i += 1
		}

		if p.ID == p2_v2.ID && p.GetVersionName() == p2_v2.GetVersionName() && p.CompositePrimaryKey == fmt.Sprintf("%d%s%s", p2_v2.ID, resource.CompositePrimaryKeySeparator, p2_v2.GetVersionName()) {
			i += 1
		}
	}

	if i != 2 {
		t.Error("valuer doesn't return correct version of products")
	}
}

// By default, qor publish2 select records by MAX(version_priority). to make it work with older version user need to define its own valuer
func TestValuer_WithVersionWithNotMaxVersionPriority(t *testing.T) {
	db := testutils.TestDB()
	productsMeta := setupProductWithVersionMeta(t, db)
	productsMeta.Valuer = func(value interface{}, ctx *qor.Context) interface{} {
		coll := value.(*CollectionWithVersion)
		if err := ctx.GetDB().Set("publish:version:mode", "multiple").Preload("Products").Find(coll).Error; err == nil {
			prods := []ProductWithVersion{}
			for _, p := range coll.Products {
				p.CompositePrimaryKeyField.CompositePrimaryKey = resource.GenCompositePrimaryKey(p.ID, p.GetVersionName())
				prods = append(prods, p)
			}
			return prods
		}

		return ""
	}

	p1 := ProductWithVersion{Name: "p1"}
	p2_v1 := ProductWithVersion{Name: "p2"}
	testutils.AssertNoErr(t, db.Save(&p1).Error)
	testutils.AssertNoErr(t, db.Save(&p2_v1).Error)
	p2_v2 := ProductWithVersion{Name: "p2"}
	p2_v2.ID = p2_v1.ID
	testutils.AssertNoErr(t, db.Save(&p2_v2).Error)
	// Here v3 has the MAX version_priority but the collection is linked with v2
	p2_v3 := ProductWithVersion{Name: "p3"}
	p2_v3.ID = p2_v1.ID
	testutils.AssertNoErr(t, db.Save(&p2_v3).Error)

	record := CollectionWithVersion{Name: "test"}
	record.Products = []ProductWithVersion{p1, p2_v2}
	testutils.AssertNoErr(t, db.Save(&record).Error)
	ctx := &qor.Context{DB: db}

	coll := CollectionWithVersion{}
	testutils.AssertNoErr(t, db.Find(&coll).Error)
	result := productsMeta.Valuer(&coll, ctx)

	associatedProducts := result.([]ProductWithVersion)
	if len(associatedProducts) != 2 {
		t.Error("valuer doesn't return correct value")
	}

	i := 0
	for _, p := range associatedProducts {
		if p.ID == p1.ID && p.CompositePrimaryKey == fmt.Sprintf("%d%s%s", p1.ID, resource.CompositePrimaryKeySeparator, p1.GetVersionName()) {
			i += 1
		}

		if p.ID == p2_v2.ID && p.GetVersionName() == p2_v2.GetVersionName() && p.CompositePrimaryKey == fmt.Sprintf("%d%s%s", p2_v2.ID, resource.CompositePrimaryKeySeparator, p2_v2.GetVersionName()) {
			i += 1
		}
	}

	if i != 2 {
		t.Error("valuer doesn't return correct version of products")
	}
}

func TestManyToManyRelation_WithVersion(t *testing.T) {
	db := testutils.TestDB()
	productsMeta := setupProductWithVersionMeta(t, db)

	p1 := ProductWithVersion{Name: "p1"}
	p2_v1 := ProductWithVersion{Name: "p2"}
	testutils.AssertNoErr(t, db.Save(&p1).Error)
	testutils.AssertNoErr(t, db.Save(&p2_v1).Error)
	p2_v2 := ProductWithVersion{Name: "p2"}
	p2_v2.ID = p2_v1.ID
	testutils.AssertNoErr(t, db.Save(&p2_v2).Error)

	record := CollectionWithVersion{Name: "test"}
	testutils.AssertNoErr(t, db.Save(&record).Error)
	ctx := &qor.Context{DB: db}
	metaValue := &resource.MetaValue{Name: productsMeta.Name, Value: []string{
		fmt.Sprintf("%d%s%s", p1.ID, resource.CompositePrimaryKeySeparator, p1.GetVersionName()),
		fmt.Sprintf("%d%s%s", p2_v2.ID, resource.CompositePrimaryKeySeparator, p2_v2.GetVersionName()),
	}}

	productsMeta.Setter(&record, metaValue, ctx)

	testutils.AssertNoErr(t, db.Preload("Products").Find(&record).Error)
	if len(record.Products) != 2 {
		t.Error("products not set to collection")
	}

	hasCorrectVersion := false
	for _, p := range record.Products {
		if p.ID == p2_v2.ID && p.GetVersionName() == p2_v2.VersionName {
			hasCorrectVersion = true
		}
	}

	if !hasCorrectVersion {
		t.Error("p2 is not associated with collection with correct version")
	}
}

func TestBelongsToRelation(t *testing.T) {
	db := testutils.TestDB()
	testutils.ResetDBTables(db, &Collection{}, &Product{}, &Tag{})

	adm := admin.New(&qor.Config{DB: db.Set(publish2.ScheduleMode, publish2.ModeOff)})
	c := adm.AddResource(&Collection{})

	tagMeta := resource.Meta{
		Name:         "Tag",
		FieldName:    "Tag",
		BaseResource: c,
	}

	var scope = &gorm.Scope{Value: c.Value}
	var getField = func(fields []*gorm.StructField, name string) *gorm.StructField {
		for _, field := range fields {
			if field.Name == name || field.DBName == name {
				return field
			}
		}
		return nil
	}

	tagMeta.FieldStruct = getField(scope.GetStructFields(), tagMeta.FieldName)

	if err := tagMeta.Initialize(); err != nil {
		t.Fatal(err)
	}

	t1 := Tag{Name: "t1"}
	testutils.AssertNoErr(t, db.Save(&t1).Error)

	record := Collection{Name: "test"}
	testutils.AssertNoErr(t, db.Save(&record).Error)
	ctx := &qor.Context{DB: db}
	metaValue := &resource.MetaValue{Name: tagMeta.Name, Value: []string{fmt.Sprintf("%d", t1.ID)}}

	tagMeta.Setter(&record, metaValue, ctx)
	testutils.AssertNoErr(t, db.Save(&record).Error)

	if record.Tag.ID != t1.ID {
		t.Error("tag not set to collection")
	}
}

func TestBelongsToWithVersionRelation(t *testing.T) {
	db := testutils.TestDB()
	registerVersionNameCallback(db)
	testutils.ResetDBTables(db, &CollectionWithVersion{}, &ProductWithVersion{}, &Manager{})

	adm := admin.New(&qor.Config{DB: db.Set(publish2.ScheduleMode, publish2.ModeOff)})
	c := adm.AddResource(&CollectionWithVersion{})

	managerMeta := resource.Meta{
		Name:         "Manager",
		FieldName:    "Manager",
		BaseResource: c,
	}

	var scope = &gorm.Scope{Value: c.Value}
	var getField = func(fields []*gorm.StructField, name string) *gorm.StructField {
		for _, field := range fields {
			if field.Name == name || field.DBName == name {
				return field
			}
		}
		return nil
	}

	managerMeta.FieldStruct = getField(scope.GetStructFields(), managerMeta.FieldName)
	if err := managerMeta.Initialize(); err != nil {
		t.Fatal(err)
	}

	m1 := Manager{Name: "Manager1"}
	testutils.AssertNoErr(t, db.Save(&m1).Error)

	record := CollectionWithVersion{Name: "test"}
	testutils.AssertNoErr(t, db.Save(&record).Error)
	ctx := &qor.Context{DB: db}
	metaValue := &resource.MetaValue{Name: managerMeta.Name, Value: []string{resource.GenCompositePrimaryKey(m1.ID, m1.GetVersionName())}}

	managerMeta.Setter(&record, metaValue, ctx)
	// Setter only sets Manager to record, we need save it explicitly in the test.
	testutils.AssertNoErr(t, db.Save(&record).Error)

	if record.ManagerID != m1.ID || record.ManagerVersionName != m1.GetVersionName() {
		t.Error("manager not set to collection")
	}
}

func registerVersionNameCallback(db *gorm.DB) {
	db.Callback().Create().Before("gorm:begin_transaction").Register("publish2:versions", func(scope *gorm.Scope) {
		if field, ok := scope.FieldByName("VersionName"); ok {
			if !field.IsBlank {
				return
			}

			name := time.Now().Format("2006-01-02")

			idField, _ := scope.FieldByName("ID")
			id := idField.Field.Interface().(uint)

			var count int
			scope.DB().Table(scope.TableName()).Unscoped().Scopes(WithoutVersion).Where("id = ? AND version_name like ?", id, name+"%").Count(&count)

			versionName := fmt.Sprintf("%s-v%v", name, count+1)
			field.Set(versionName)
		}
	})

	db.Callback().Create().After("gorm:begin_transaction").Register("publish2:version_priority", updateVersionPriority())
}

//  Test assigning associations when creating new version. the associations should assign to correct version after save
func TestAssigningAssociationsOnNewVersion(t *testing.T) {
	db := testutils.TestDB()
	productsMeta := setupProductWithVersionMeta(t, db)

	p1 := ProductWithVersion{Name: "p1"}
	p2_v1 := ProductWithVersion{Name: "p2"}
	testutils.AssertNoErr(t, db.Save(&p1).Error)
	testutils.AssertNoErr(t, db.Save(&p2_v1).Error)
	p2_v2 := ProductWithVersion{Name: "p2"}
	p2_v2.ID = p2_v1.ID
	testutils.AssertNoErr(t, db.Save(&p2_v2).Error)

	record := CollectionWithVersion{Name: "test"}
	testutils.AssertNoErr(t, db.Save(&record).Error)

	newVersionCollection := CollectionWithVersion{Name: "test-v2"}
	newVersionCollection.ID = record.ID

	formValues := map[string][]string{"QorResource.VersionName": {}}
	ctx := &qor.Context{DB: db, Request: &http.Request{Form: formValues}}
	metaValue := &resource.MetaValue{Name: productsMeta.Name, Value: []string{
		fmt.Sprintf("%d%s%s", p1.ID, resource.CompositePrimaryKeySeparator, p1.GetVersionName()),
		fmt.Sprintf("%d%s%s", p2_v2.ID, resource.CompositePrimaryKeySeparator, p2_v2.GetVersionName()),
	}}

	productsMeta.Setter(&newVersionCollection, metaValue, ctx)
	// For new version, the object will not be saved inside setter, so we have to call save explicitly
	testutils.AssertNoErr(t, db.Save(&newVersionCollection).Error)

	testutils.AssertNoErr(t, db.Preload("Products").Find(&newVersionCollection).Error)
	if len(newVersionCollection.Products) != 2 {
		t.Error("products not set to collection")
	}

	hasCorrectVersion := false
	for _, p := range newVersionCollection.Products {
		if p.ID == p2_v2.ID && p.GetVersionName() == p2_v2.VersionName {
			hasCorrectVersion = true
		}
	}

	if !hasCorrectVersion {
		t.Error("p2 is not associated with collection with correct version")
	}
}
func TestSwitchRecordToNewVersionIfNeeded(t *testing.T) {
	db := testutils.TestDB()
	testutils.ResetDBTables(db, &CollectionWithVersion{})
	registerVersionNameCallback(db)

	record := CollectionWithVersion{Name: "test"}
	testutils.AssertNoErr(t, db.Save(&record).Error)
	oldVersionName := record.VersionName

	formValues := map[string][]string{"QorResource.VersionName": {}}
	ctx := &qor.Context{DB: db, Request: &http.Request{Form: formValues}}

	newRecord := resource.SwitchRecordToNewVersionIfNeeded(ctx, record)

	if newRecord.(CollectionWithVersion).VersionName == oldVersionName {
		t.Error("new version name is not assigned to record")
	}
}
func TestSwitchRecordToNewVersionIfNeeded_EditExistingVersion(t *testing.T) {
	db := testutils.TestDB()
	testutils.ResetDBTables(db, &CollectionWithVersion{})
	registerVersionNameCallback(db)

	record := CollectionWithVersion{Name: "test"}
	testutils.AssertNoErr(t, db.Save(&record).Error)
	oldVersionName := record.VersionName

	formValues := map[string][]string{"QorResource.VersionName": []string{oldVersionName}}
	ctx := &qor.Context{DB: db, Request: &http.Request{Form: formValues}}

	newRecord := resource.SwitchRecordToNewVersionIfNeeded(ctx, record)

	if newRecord.(CollectionWithVersion).VersionName != oldVersionName {
		t.Error("new version name is assigned to record when it shouldn't ")
	}
}

type Athlete struct {
	gorm.Model

	publish2.Version
	publish2.Schedule

	Name string
}

func TestSwitchRecordToNewVersionIfNeeded_WithNoAssignVersionMethod(t *testing.T) {
	db := testutils.TestDB()
	testutils.ResetDBTables(db, &Athlete{})
	registerVersionNameCallback(db)

	record := Athlete{Name: "test"}
	testutils.AssertNoErr(t, db.Save(&record).Error)
	oldVersionName := record.VersionName

	formValues := map[string][]string{"QorResource.VersionName": {}}
	ctx := &qor.Context{DB: db, Request: &http.Request{Form: formValues}}

	newRecord := resource.SwitchRecordToNewVersionIfNeeded(ctx, record)

	if newRecord.(Athlete).VersionName != oldVersionName {
		t.Error("new version name is assigned to record")
	}
}
