package conventions_test

const (
	moduleRoot        = "."
	goExtension       = ".go"
	testFileSuffix    = "_test.go"
	constantsFile     = "constants.go"
	constantsTestFile = "constants_test.go"
	emptyString       = `""`
	literalZero       = "0"
	literalOne        = "1"
	tableNameKey      = "name"
	subtestMethod     = "Run"
	subtestArguments  = 2
	modulePath        = "github.com/craig-hunt/go-standards"
	domainRootFormat  = "internal/%s/"
	importSeparator   = "/"
	literalFormat     = "%s: literal %s belongs in a named constant"
	grabBagFormat     = "%s: package %q is named for its role rather than what it does"
	importFormat      = "%s: domain package imports %s"
)

var (
	skippedDirectories = []string{".git", "testdata", "node_modules", "bin"}
	grabBagNames       = []string{"util", "utils", "common", "helper", "helpers", "misc", "shared", "base"}
	domainPackages     = []string{"task", "signup", "inventory"}
	infrastructure     = []string{modulePath + "/internal/postgres", modulePath + "/internal/api", "github.com/jackc/pgx"}
)
