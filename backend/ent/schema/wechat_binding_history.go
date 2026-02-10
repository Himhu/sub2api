package schema

import (
	"time"

	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// WeChatBindingHistory holds tombstone records of past WeChat bindings.
type WeChatBindingHistory struct {
	ent.Schema
}

func (WeChatBindingHistory) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "wechat_binding_history"},
	}
}

func (WeChatBindingHistory) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (WeChatBindingHistory) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("app_id").
			MaxLen(64).
			NotEmpty(),
		field.String("openid").
			MaxLen(64).
			NotEmpty(),
		field.Time("unbound_at").
			Default(time.Now),
		field.String("reason").
			MaxLen(64).
			Default("user_unbind"),
	}
}

func (WeChatBindingHistory) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("app_id", "openid").
			Unique(),
		index.Fields("user_id"),
	}
}
