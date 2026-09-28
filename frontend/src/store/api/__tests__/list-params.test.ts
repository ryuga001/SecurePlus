import { describe, expect, it } from "vitest";

import { listParams } from "@/store/api/list-params";

describe("listParams", () => {
  it("maps page and pageSize to snake_case", () => {
    expect(listParams({ page: 2, pageSize: 25, filters: {} })).toEqual({
      page: 2,
      page_size: 25,
    });
  });

  it("includes sort fields only when a sort key is present", () => {
    const withSort = listParams({ page: 1, pageSize: 10, sortBy: "created_at", sortDir: "desc", filters: {} });
    expect(withSort).toMatchObject({ sort_by: "created_at", sort_dir: "desc" });

    const withoutSort = listParams({ page: 1, pageSize: 10, filters: {} });
    expect(withoutSort).not.toHaveProperty("sort_by");
    expect(withoutSort).not.toHaveProperty("sort_dir");
  });

  it("spreads filters alongside pagination", () => {
    const params = listParams({
      page: 1,
      pageSize: 10,
      filters: { search: "acme", role_id: "7" },
    });

    expect(params).toEqual({ page: 1, page_size: 10, search: "acme", role_id: "7" });
  });
});
