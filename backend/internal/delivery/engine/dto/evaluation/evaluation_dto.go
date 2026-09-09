package evaluation

import (
	"context"
	"regexp"
)

type MessageContext struct {
	CorrelationID string
	MessageID     string
	CustomerID    int
	ConfigID      int
	From          string
	SenderDomain  string
	Recipients    []string
	Raw           []byte
}

type Attachment struct {
	Filename    string
	Extension   string
	ContentType string
	Size        int64
}

type ContentPart struct {
	Location string
	Text     string
}

type ParsedMessage struct {
	Subject     string
	Parts       []ContentPart
	Attachments []Attachment
}

type Restriction struct {
	Mode   string
	Values []string
}

type RuleRecord struct {
	PolicyID   int
	PolicyName string
	Action     string
	RuleID     int
	RuleName   string
	RuleType   string
	RuleValue  string
}

type PolicyRecord struct {
	PolicyID              int
	PolicyName            string
	Action                string
	DomainRestriction     Restriction
	AttachmentRestriction Restriction
}

type PolicySet struct {
	CustomerID  int
	EmailUserID int
	Policies    []PolicyRecord
	Rules       []RuleRecord
}

type RestrictionSet struct {
	Blocked map[string]PolicyRef
	Allowed map[string]PolicyRef
}

type PolicyRef struct {
	PolicyID   int
	PolicyName string
}

type EffectiveRestrictions struct {
	Domain     RestrictionSet
	Attachment RestrictionSet
}

type CompiledKeyword struct {
	Rule       RuleRecord
	Normalized string
}

type CompiledRegex struct {
	Rule    RuleRecord
	Pattern *regexp.Regexp
}

type CompiledRules struct {
	Keywords  []CompiledKeyword
	Automaton Automaton
	Regexes   []CompiledRegex
}

type CompiledSet struct {
	CustomerID   int
	EmailUserID  int
	PolicyCount  int
	Restrictions EffectiveRestrictions
	Rules        CompiledRules
	RuleTypes    []string
}

type Automaton interface {
	Find(text string) []AutomatonHit
}

type AutomatonHit struct {
	Index int
	Start int
	End   int
}

type RuleMatch struct {
	PolicyID        int
	PolicyName      string
	Action          string
	RuleID          int
	RuleName        string
	RuleType        string
	ConfiguredValue string
	Occurrences     int
	Locations       []string
}

type RestrictionViolation struct {
	Kind        string
	Mode        string
	Value       string
	Recipient   string
	Filename    string
	ContentType string
	PolicyID    int
	PolicyName  string
}

type WithheldRecipient struct {
	Email      string
	Domain     string
	PolicyID   int
	PolicyName string
}

type MatchInput struct {
	Parts []ContentPart
}

type EvaluationResult struct {
	CorrelationID         string
	CustomerID            int
	EmailUserID           int
	Decision              string
	Trigger               string
	EffectiveAction       string
	EvaluatedPolicyCount  int
	TriggeredPolicyIDs    []int
	Recipients            []string
	Withheld              []WithheldRecipient
	RestrictionViolations []RestrictionViolation
	Matches               []RuleMatch
}

type ActionRequest struct {
	CorrelationID   string
	CustomerID      int
	Action          string
	Trigger         string
	Recipients      []string
	TriggeredPolicy []int
}

type ActionResult struct {
	Action string
	Status string
	Error  string
}

type Outcome struct {
	Result    EvaluationResult
	Withheld  []WithheldRecipient
	Delivered []string
}

type MessageParser interface {
	Parse(raw []byte) (ParsedMessage, error)
}

type PolicyCache interface {
	Load(ctx context.Context, customerID int, sender string) (*CompiledSet, error)
}

type Aggregator interface {
	Aggregate(policies []PolicyRecord) EffectiveRestrictions
}

type DomainRestrictionEvaluator interface {
	Evaluate(recipients []string, restrictions EffectiveRestrictions) []RestrictionViolation
}

type AttachmentRestrictionEvaluator interface {
	Evaluate(attachments []Attachment, restrictions EffectiveRestrictions) []RestrictionViolation
}

type RuleMatcher interface {
	Type() string
	Match(input MatchInput, compiled CompiledRules) []RuleMatch
}

type MatcherFactory interface {
	For(ruleType string) (RuleMatcher, bool)
}

type ContentEngine interface {
	Evaluate(input MatchInput, compiled CompiledSet) []RuleMatch
}

type ActionResolver interface {
	Resolve(matches []RuleMatch) string
}

type ActionExecutor interface {
	Action() string
	Execute(ctx context.Context, request ActionRequest) (ActionResult, error)
}

type ActionFactory interface {
	For(action string) (ActionExecutor, bool)
}

type IncidentRecorder interface {
	Record(ctx context.Context, result EvaluationResult, message MessageContext) error
	UpdateAction(ctx context.Context, correlationID string, result ActionResult) error
}

type Enforcer interface {
	Enforce(ctx context.Context, message MessageContext) (Outcome, error)
}
