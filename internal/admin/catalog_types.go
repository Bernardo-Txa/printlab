package admin

type AdminFieldErrors map[string]string

func (e AdminFieldErrors) Get(field string) string {
	if e == nil {
		return ""
	}

	return e[field]
}

func (e AdminFieldErrors) Add(field string, message string) {
	if e == nil {
		return
	}
	if _, exists := e[field]; !exists {
		e[field] = message
	}
}

func (e AdminFieldErrors) Any() bool {
	return len(e) > 0
}

type AdminFilterOption struct {
	Value  string
	Label  string
	URL    string
	Active bool
}

type AdminSelectOption struct {
	ID       string
	Label    string
	Active   bool
	Selected bool
}

type AdminProductListFilter struct {
	Status string
	Query  string
}

type AdminProductListPage struct {
	Products []AdminProductListItem
	Status   string
	Query    string
	Options  []AdminFilterOption
}

type AdminProductListItem struct {
	ID                 string
	Name               string
	Slug               string
	CategoryName       string
	CategoryActive     bool
	CategoryLabel      string
	PriceCents         int64
	PriceBRL           string
	IsActive           bool
	StatusLabel        string
	IsFeatured         bool
	FeaturedLabel      string
	VariantCount       int
	VariantCountLabel  string
	HasShippingProfile bool
	ShippingLabel      string
	Warnings           []string
	DetailURL          string
	VariantsURL        string
}

type AdminProductHeader struct {
	ID        string
	Name      string
	Slug      string
	DetailURL string
}

type AdminProductForm struct {
	Name             string
	Slug             string
	CategoryID       string
	ShortDescription string
	Description      string
	PriceBRL         string
	IsActive         bool
	IsFeatured       bool
	UseShipping      bool
	ShippingWeightG  string
	ShippingHeightMM string
	ShippingWidthMM  string
	ShippingLengthMM string
}

type AdminProductFormPage struct {
	Title       string
	Action      string
	SubmitLabel string
	BackURL     string
	IsNew       bool
	Form        AdminProductForm
	Errors      AdminFieldErrors
	Categories  []AdminSelectOption
	Variants    []AdminVariantListItem
	Message     string
}

type AdminCategoryListPage struct {
	Categories []AdminCategoryListItem
}

type AdminCategoryListItem struct {
	ID          string
	Name        string
	Slug        string
	Description string
	IsActive    bool
	StatusLabel string
	DetailURL   string
}

type AdminCategoryForm struct {
	Name        string
	Slug        string
	Description string
	IsActive    bool
}

type AdminCategoryFormPage struct {
	Title       string
	Action      string
	SubmitLabel string
	BackURL     string
	IsNew       bool
	Form        AdminCategoryForm
	Errors      AdminFieldErrors
	Message     string
}

type AdminVariantListItem struct {
	ID                 string
	Name               string
	Slug               string
	SKU                string
	PriceLabel         string
	IsActive           bool
	StatusLabel        string
	IsDefault          bool
	DefaultLabel       string
	SortOrder          int
	PrintTimeLabel     string
	HasShippingProfile bool
	ShippingLabel      string
	RecipeCount        int
	RecipeWeightMg     int64
	RecipeWeightLabel  string
	DetailURL          string
}

type AdminVariantForm struct {
	Name             string
	Slug             string
	SKU              string
	PriceBRL         string
	IsActive         bool
	IsDefault        bool
	SortOrder        string
	PrintTimeMinutes string
	UseShipping      bool
	ShippingWeightG  string
	ShippingHeightMM string
	ShippingWidthMM  string
	ShippingLengthMM string
}

type AdminVariantFormPage struct {
	Title           string
	Action          string
	SubmitLabel     string
	BackURL         string
	IsNew           bool
	Product         AdminProductHeader
	VariantID       string
	Form            AdminVariantForm
	Errors          AdminFieldErrors
	Recipe          []AdminRecipeComponent
	RecipeForm      AdminRecipeForm
	RecipeErrors    AdminFieldErrors
	MaterialOptions []AdminSelectOption
	ColorOptions    []AdminSelectOption
	Message         string
}

type AdminRecipeComponent struct {
	ID                   string
	MaterialID           string
	MaterialName         string
	MaterialActive       bool
	MaterialLabel        string
	ColorID              string
	ColorName            string
	ColorHex             string
	ColorActive          bool
	ColorLabel           string
	EstimatedWeightMg    int64
	EstimatedWeightInput string
	EstimatedWeightLabel string
	Label                string
	SortOrder            int
	SortOrderInput       string
	UpdateAction         string
	RemoveAction         string
}

type AdminRecipeForm struct {
	MaterialID      string
	ColorID         string
	EstimatedWeight string
	Label           string
	SortOrder       string
}

type AdminMaterialListPage struct {
	Materials []AdminMaterialListItem
}

type AdminMaterialListItem struct {
	ID          string
	Name        string
	Slug        string
	Description string
	IsActive    bool
	StatusLabel string
	DetailURL   string
}

type AdminMaterialForm struct {
	Name        string
	Slug        string
	Description string
	IsActive    bool
}

type AdminMaterialFormPage struct {
	Title       string
	Action      string
	SubmitLabel string
	BackURL     string
	IsNew       bool
	Form        AdminMaterialForm
	Errors      AdminFieldErrors
	Message     string
}

type AdminColorListPage struct {
	Colors []AdminColorListItem
}

type AdminColorListItem struct {
	ID          string
	Name        string
	Slug        string
	HexColor    string
	IsActive    bool
	StatusLabel string
	DetailURL   string
}

type AdminColorForm struct {
	Name     string
	Slug     string
	HexColor string
	IsActive bool
}

type AdminColorFormPage struct {
	Title       string
	Action      string
	SubmitLabel string
	BackURL     string
	IsNew       bool
	Form        AdminColorForm
	Errors      AdminFieldErrors
	Message     string
}

type AdminBoxListPage struct {
	Boxes []AdminBoxListItem
}

type AdminBoxListItem struct {
	ID               string
	Name             string
	Slug             string
	InternalLabel    string
	ExternalLabel    string
	PackagingWeightG int
	WeightLabel      string
	IsActive         bool
	StatusLabel      string
	SortOrder        int
	DetailURL        string
}

type AdminBoxForm struct {
	Name             string
	Slug             string
	InternalHeightMM string
	InternalWidthMM  string
	InternalLengthMM string
	ExternalHeightMM string
	ExternalWidthMM  string
	ExternalLengthMM string
	PackagingWeightG string
	IsActive         bool
	SortOrder        string
}

type AdminBoxFormPage struct {
	Title       string
	Action      string
	SubmitLabel string
	BackURL     string
	IsNew       bool
	Form        AdminBoxForm
	Errors      AdminFieldErrors
	Message     string
}

type AdminShippingProfile struct {
	WeightG  int64
	HeightMM int
	WidthMM  int
	LengthMM int
}

type AdminProductSaveInput struct {
	ID               string
	Name             string
	Slug             string
	CategoryID       string
	ShortDescription string
	Description      string
	PriceCents       int64
	IsActive         bool
	IsFeatured       bool
	ShippingProfile  *AdminShippingProfile
}

type AdminCategorySaveInput struct {
	ID          string
	Name        string
	Slug        string
	Description string
	IsActive    bool
}

type AdminVariantSaveInput struct {
	ID               string
	ProductID        string
	Name             string
	Slug             string
	SKU              string
	PriceCents       *int64
	IsActive         bool
	IsDefault        bool
	SortOrder        int
	PrintTimeMinutes *int
	ShippingProfile  *AdminShippingProfile
}

type AdminRecipeSaveInput struct {
	ID                string
	ProductID         string
	VariantID         string
	MaterialID        string
	ColorID           string
	EstimatedWeightMg int64
	Label             string
	SortOrder         int
}

type AdminMaterialSaveInput struct {
	ID          string
	Name        string
	Slug        string
	Description string
	IsActive    bool
}

type AdminColorSaveInput struct {
	ID       string
	Name     string
	Slug     string
	HexColor string
	IsActive bool
}

type AdminBoxSaveInput struct {
	ID               string
	Name             string
	Slug             string
	InternalHeightMM int
	InternalWidthMM  int
	InternalLengthMM int
	ExternalHeightMM int
	ExternalWidthMM  int
	ExternalLengthMM int
	PackagingWeightG int
	IsActive         bool
	SortOrder        int
}
