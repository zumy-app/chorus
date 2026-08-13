// PayPal Billing subscription plans. Plan IDs come from the PayPal dashboard;
// both are daily-rate subscriptions that the backend grants entitlements for.
export const PAYPAL_MONTHLY_URL =
  'https://www.paypal.com/webapps/billing/plans/subscribe?plan_id=P-9EF672607G749152JNJ6SZ3Y'
export const PAYPAL_YEARLY_URL =
  'https://www.paypal.com/webapps/billing/plans/subscribe?plan_id=P-0PU22435FL441062UNJ6S2XY'

// Display prices (USD). Annual = 12 × $7.99 list, discounted to 2 months free.
export const MONTHLY_PRICE = '$7.99'
export const YEARLY_PRICE = '$79.90'
export const YEARLY_LIST_PRICE = '$95.88'