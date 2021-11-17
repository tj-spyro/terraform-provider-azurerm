package eventsubscriptions

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
)

type ListRegionalByResourceGroupResponse struct {
	HttpResponse *http.Response
	Model        *[]EventSubscription

	nextLink     *string
	nextPageFunc func(ctx context.Context, nextLink string) (ListRegionalByResourceGroupResponse, error)
}

type ListRegionalByResourceGroupCompleteResult struct {
	Items []EventSubscription
}

func (r ListRegionalByResourceGroupResponse) HasMore() bool {
	return r.nextLink != nil
}

func (r ListRegionalByResourceGroupResponse) LoadMore(ctx context.Context) (resp ListRegionalByResourceGroupResponse, err error) {
	if !r.HasMore() {
		err = fmt.Errorf("no more pages returned")
		return
	}
	return r.nextPageFunc(ctx, *r.nextLink)
}

type ListRegionalByResourceGroupOptions struct {
	Filter *string
	Top    *int64
}

func DefaultListRegionalByResourceGroupOptions() ListRegionalByResourceGroupOptions {
	return ListRegionalByResourceGroupOptions{}
}

func (o ListRegionalByResourceGroupOptions) toQueryString() map[string]interface{} {
	out := make(map[string]interface{})

	if o.Filter != nil {
		out["$filter"] = *o.Filter
	}

	if o.Top != nil {
		out["$top"] = *o.Top
	}

	return out
}

// ListRegionalByResourceGroup ...
func (c EventSubscriptionsClient) ListRegionalByResourceGroup(ctx context.Context, id ProviderLocationId, options ListRegionalByResourceGroupOptions) (resp ListRegionalByResourceGroupResponse, err error) {
	req, err := c.preparerForListRegionalByResourceGroup(ctx, id, options)
	if err != nil {
		err = autorest.NewErrorWithError(err, "eventsubscriptions.EventSubscriptionsClient", "ListRegionalByResourceGroup", nil, "Failure preparing request")
		return
	}

	resp.HttpResponse, err = c.Client.Send(req, azure.DoRetryWithRegistration(c.Client))
	if err != nil {
		err = autorest.NewErrorWithError(err, "eventsubscriptions.EventSubscriptionsClient", "ListRegionalByResourceGroup", resp.HttpResponse, "Failure sending request")
		return
	}

	resp, err = c.responderForListRegionalByResourceGroup(resp.HttpResponse)
	if err != nil {
		err = autorest.NewErrorWithError(err, "eventsubscriptions.EventSubscriptionsClient", "ListRegionalByResourceGroup", resp.HttpResponse, "Failure responding to request")
		return
	}
	return
}

// ListRegionalByResourceGroupComplete retrieves all of the results into a single object
func (c EventSubscriptionsClient) ListRegionalByResourceGroupComplete(ctx context.Context, id ProviderLocationId, options ListRegionalByResourceGroupOptions) (ListRegionalByResourceGroupCompleteResult, error) {
	return c.ListRegionalByResourceGroupCompleteMatchingPredicate(ctx, id, options, EventSubscriptionPredicate{})
}

// ListRegionalByResourceGroupCompleteMatchingPredicate retrieves all of the results and then applied the predicate
func (c EventSubscriptionsClient) ListRegionalByResourceGroupCompleteMatchingPredicate(ctx context.Context, id ProviderLocationId, options ListRegionalByResourceGroupOptions, predicate EventSubscriptionPredicate) (resp ListRegionalByResourceGroupCompleteResult, err error) {
	items := make([]EventSubscription, 0)

	page, err := c.ListRegionalByResourceGroup(ctx, id, options)
	if err != nil {
		err = fmt.Errorf("loading the initial page: %+v", err)
		return
	}
	if page.Model != nil {
		for _, v := range *page.Model {
			if predicate.Matches(v) {
				items = append(items, v)
			}
		}
	}

	for page.HasMore() {
		page, err = page.LoadMore(ctx)
		if err != nil {
			err = fmt.Errorf("loading the next page: %+v", err)
			return
		}

		if page.Model != nil {
			for _, v := range *page.Model {
				if predicate.Matches(v) {
					items = append(items, v)
				}
			}
		}
	}

	out := ListRegionalByResourceGroupCompleteResult{
		Items: items,
	}
	return out, nil
}

// preparerForListRegionalByResourceGroup prepares the ListRegionalByResourceGroup request.
func (c EventSubscriptionsClient) preparerForListRegionalByResourceGroup(ctx context.Context, id ProviderLocationId, options ListRegionalByResourceGroupOptions) (*http.Request, error) {
	queryParameters := map[string]interface{}{
		"api-version": defaultApiVersion,
	}

	for k, v := range options.toQueryString() {
		queryParameters[k] = autorest.Encode("query", v)
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/json; charset=utf-8"),
		autorest.AsGet(),
		autorest.WithBaseURL(c.baseUri),
		autorest.WithPath(fmt.Sprintf("%s/eventSubscriptions", id.ID())),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// preparerForListRegionalByResourceGroupWithNextLink prepares the ListRegionalByResourceGroup request with the given nextLink token.
func (c EventSubscriptionsClient) preparerForListRegionalByResourceGroupWithNextLink(ctx context.Context, nextLink string) (*http.Request, error) {
	uri, err := url.Parse(nextLink)
	if err != nil {
		return nil, fmt.Errorf("parsing nextLink %q: %+v", nextLink, err)
	}
	queryParameters := map[string]interface{}{}
	for k, v := range uri.Query() {
		if len(v) == 0 {
			continue
		}
		val := v[0]
		val = autorest.Encode("query", val)
		queryParameters[k] = val
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/json; charset=utf-8"),
		autorest.AsGet(),
		autorest.WithBaseURL(c.baseUri),
		autorest.WithPath(uri.Path),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// responderForListRegionalByResourceGroup handles the response to the ListRegionalByResourceGroup request. The method always
// closes the http.Response Body.
func (c EventSubscriptionsClient) responderForListRegionalByResourceGroup(resp *http.Response) (result ListRegionalByResourceGroupResponse, err error) {
	type page struct {
		Values   []EventSubscription `json:"value"`
		NextLink *string             `json:"nextLink"`
	}
	var respObj page
	err = autorest.Respond(
		resp,
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&respObj),
		autorest.ByClosing())
	result.HttpResponse = resp
	result.Model = &respObj.Values
	result.nextLink = respObj.NextLink
	if respObj.NextLink != nil {
		result.nextPageFunc = func(ctx context.Context, nextLink string) (result ListRegionalByResourceGroupResponse, err error) {
			req, err := c.preparerForListRegionalByResourceGroupWithNextLink(ctx, nextLink)
			if err != nil {
				err = autorest.NewErrorWithError(err, "eventsubscriptions.EventSubscriptionsClient", "ListRegionalByResourceGroup", nil, "Failure preparing request")
				return
			}

			result.HttpResponse, err = c.Client.Send(req, azure.DoRetryWithRegistration(c.Client))
			if err != nil {
				err = autorest.NewErrorWithError(err, "eventsubscriptions.EventSubscriptionsClient", "ListRegionalByResourceGroup", result.HttpResponse, "Failure sending request")
				return
			}

			result, err = c.responderForListRegionalByResourceGroup(result.HttpResponse)
			if err != nil {
				err = autorest.NewErrorWithError(err, "eventsubscriptions.EventSubscriptionsClient", "ListRegionalByResourceGroup", result.HttpResponse, "Failure responding to request")
				return
			}

			return
		}
	}
	return
}
