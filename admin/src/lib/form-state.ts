// Shared by the Server Actions and the client forms (no server-only imports).

/** State returned by the Server Actions of the forms (useActionState). */
export type FormState = {
  error?: string;
  fields?: Record<string, string>;
  /** Submitted values, so the form keeps them after an error. */
  values?: Record<string, string>;
};

export const emptyForm: FormState = {};

/**
 * A Server Action ready for useActionState. Pages bind the record id on the
 * server and pass the result down: binding inside a client component creates
 * a new action on every render, and the form state of a submission made
 * before hydration then never matches, leaving the response hanging.
 */
export type FormAction = (state: FormState, formData: FormData) => Promise<FormState>;
