package bdspb

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAccountRPCContractNeverTrustsCallerAccountID(t *testing.T) {
	// Authentication comes from the future gRPC interceptor and its live
	// PostgreSQL session lookup, not a request-owned account_id field.
	for _, msg := range []proto.Message{
		&GetCurrentAccountRequest{},
		&ListWorkspacesRequest{},
	} {
		if got := msg.ProtoReflect().Descriptor().Fields().Len(); got != 0 {
			t.Fatalf("%s unexpectedly accepts caller-owned identity fields: %d",
				msg.ProtoReflect().Descriptor().FullName(), got)
		}
	}
	service := File_bdspro_v1_account_proto.Services().ByName(protoreflect.Name("AccountService"))
	if service == nil || service.Methods().Len() != 2 {
		t.Fatal("missing stable AccountService RPC surface")
	}
	if service.Methods().ByName("GetCurrentAccount") == nil ||
		service.Methods().ByName("ListWorkspaces") == nil {
		t.Fatal("AccountService methods do not match canonical proto contract")
	}
}

func TestWorkspaceProtobufRoundTrip(t *testing.T) {
	now := time.Date(2026, 10, 21, 9, 30, 0, 0, time.UTC)
	in := &ListWorkspacesResponse{Workspaces: []*Workspace{{
		OrganizationId: 42,
		OrganizationName: "Example Organization",
		MembershipId: 87,
		JoinedAt: timestamppb.New(now),
	}}}
	data, err := proto.Marshal(in)
	if err != nil { t.Fatal(err) }
	var decoded ListWorkspacesResponse
	if err := proto.Unmarshal(data, &decoded); err != nil { t.Fatal(err) }
	if !proto.Equal(in, &decoded) {
		t.Fatalf("Protobuf binary round-trip changed message: %v", &decoded)
	}
	if got := decoded.Workspaces[0].GetJoinedAt().AsTime(); !got.Equal(now) {
		t.Fatalf("join timestamp changed: %s", got)
	}
}
