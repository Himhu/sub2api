package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// WeChatBinding holds the schema definition for the WeChatBinding entity.
type WeChatBinding struct {
	ent.Schema
}

func (WeChatBinding) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "wechat_bindings"},
	}
}

func (WeChatBinding) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (WeChatBinding) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("app_id").
			MaxLen(64).
			NotEmpty(),
		field.String("openid").
			MaxLen(64).
			NotEmpty(),
		field.String("unionid").
			MaxLen(64).
			Optional().
			Nillable(),
		field.Bool("subscribed").
			Default(true),
	}
}

func (WeChatBinding) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("wechat_bindings").
			Unique().
			Required().
			Field("user_id"),
	}
}

func (WeChatBinding) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("app_id", "openid").
			Unique(),
		index.Edges("user").
			Unique(),
	}
}
