package mem_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/gokv/mem"
)

type String string

func (s *String) UnmarshalBinary(data []byte) error {
	*s = String(data)
	return nil
}

func (s String) MarshalBinary() (data []byte, err error) {
	return []byte(s), nil
}

func TestStore(t *testing.T) {
	type checkFunc func(*mem.Store) error
	checks := func(fns ...checkFunc) []checkFunc { return fns }

	hasValue := func(key string, want String) checkFunc {
		return func(s *mem.Store) error {
			var have String
			ok, err := s.Get(key, &have)
			if err != nil {
				return fmt.Errorf("unexpected error: `%v`", err)
			}
			if !ok {
				return fmt.Errorf("key not found: %q", key)
			}
			if have != want {
				return fmt.Errorf("expected value %q, found %q", want, have)
			}
			return nil
		}
	}
	hasNotKey := func(key string) checkFunc {
		return func(s *mem.Store) error {
			var value String
			ok, err := s.Get(key, &value)
			if err != nil {
				return fmt.Errorf("unexpected error: `%v`", err)
			}
			if ok {
				return fmt.Errorf("key %q unexpectedly found: %q", key, value)
			}
			return nil
		}
	}
	hasLength := func(want int) checkFunc {
		return func(s *mem.Store) error {
			if have := s.Length(); have != want {
				return fmt.Errorf("expected length %d, found %d", want, have)
			}
			return nil
		}
	}

	type storeBuilder func(*mem.Store)
	buildStore := func(fns ...storeBuilder) []storeBuilder { return fns }

	withValue := func(key string, v String) storeBuilder {
		return func(s *mem.Store) {
			if err := s.Set(key, v); err != nil {
				panic(err)
			}
		}
	}
	withValueTimeout := func(key string, v String, timeout time.Duration) storeBuilder {
		return func(s *mem.Store) {
			if err := s.SetWithTimeout(key, v, timeout); err != nil {
				panic(err)
			}
		}
	}
	deleteKey := func(key string) storeBuilder {
		return func(s *mem.Store) {
			s.Del(key)
		}
	}

	after := func(d time.Duration, check checkFunc) checkFunc {
		return func(s *mem.Store) error {
			time.Sleep(d)
			return check(s)
		}
	}

	for _, tc := range [...]struct {
		name          string
		storeBuilders []storeBuilder
		checks        []checkFunc
	}{
		{
			"hit",
			buildStore(
				withValue("mykey", String("somevalue")),
			),
			checks(
				hasLength(1),
				hasValue("mykey", String("somevalue")),
			),
		},
		{
			"miss",
			buildStore(),
			checks(
				hasLength(0),
				hasNotKey("unset key"),
			),
		},
		{
			"store 3, get lenght 3",
			buildStore(
				withValue("key1", String(")=IM()=UNY(Hf09riècg,àrgò")),
				withValue("key2", String("somevalue")),
			),
			checks(
				hasLength(2),
				hasValue("key1", String(")=IM()=UNY(Hf09riècg,àrgò")),
				hasValue("key2", String("somevalue")),
			),
		},
		{
			"reads before timeout",
			buildStore(
				withValueTimeout("volatile key", String("somevalue"), time.Second),
			),
			checks(
				hasLength(1),
				hasValue("volatile key", String("somevalue")),
			),
		},
		{
			"misses after timeout",
			buildStore(
				withValueTimeout("volatile key", String("somevalue"), time.Millisecond),
			),
			checks(
				after(
					time.Millisecond,
					hasNotKey("volatile key"),
				),
			),
		},
		{
			"deletes an entry",
			buildStore(
				withValue("mykey", String("some value")),
				deleteKey("mykey"),
			),
			checks(
				hasNotKey("mykey"),
			),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := mem.New()

			for _, build := range tc.storeBuilders {
				build(s)
			}

			for _, check := range tc.checks {
				if e := check(s); e != nil {
					t.Error(e)
				}
			}
		})
	}
}