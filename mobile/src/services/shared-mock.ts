/**
 * Shared jest.fn for the @chorus/shared mock factory. Kept in its own module
 * so the (hoisted) jest.mock factory can reference it — variables used inside
 * a factory must either be prefixed with `mock` or live in another module.
 */
export const mockLogin = jest.fn();
