// The model form's request and the errors it shows. The server validates the
// form against PicoClaw's provider table: it fills in a provider's default API
// base, knows which local servers need no key and names the field to correct.
// The browser only insists on the identifier, so it never rejects a form the
// server would accept.

export type ModelConfigField = 'model' | 'api_base' | 'api_key';

export type ModelConfigError = {
  message: string;
  field?: ModelConfigField;
};

export type ModelConfigInput = {
  model: string;
  apiBase: string;
  apiKey: string;
};

export type ModelConfigRequest = {
  model: string;
  api_base: string;
  api_key: string;
};

const fields: readonly ModelConfigField[] = ['model', 'api_base', 'api_key'];

export function toModelConfigRequest(input: ModelConfigInput): ModelConfigRequest {
  return {
    model: input.model.trim(),
    api_base: input.apiBase.trim(),
    api_key: input.apiKey.trim()
  };
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function nonEmptyString(value: unknown) {
  return typeof value === 'string' && value.trim() !== '' ? value.trim() : undefined;
}

function fieldOf(data: unknown): ModelConfigField | undefined {
  if (!isObject(data)) {
    return undefined;
  }
  const field = data.field;
  return fields.find((candidate) => candidate === field);
}

// A failed save as returned by the API: {code, message, data: {field}}.
export function modelConfigErrorFromResponse(
  response: unknown,
  fallback: string
): ModelConfigError {
  if (!isObject(response)) {
    return { message: fallback };
  }
  return {
    message: nonEmptyString(response.message) ?? nonEmptyString(response.msg) ?? fallback,
    field: fieldOf(response.data)
  };
}

// A save that did not produce an API response body with code 0: an HTTP
// error (whose body may still be an API error), a timeout or a lost network.
export function modelConfigErrorFromException(error: unknown, fallback: string): ModelConfigError {
  if (isObject(error) && isObject(error.response) && isObject(error.response.data)) {
    const fromBody = modelConfigErrorFromResponse(error.response.data, '');
    if (fromBody.message) {
      return fromBody;
    }
  }
  if (error instanceof Error && error.message.trim() !== '') {
    return { message: error.message };
  }
  return { message: fallback };
}
