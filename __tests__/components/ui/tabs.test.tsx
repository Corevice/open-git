import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";

describe("Tabs", () => {
  it("shows the default panel and switches on trigger click", async () => {
    const user = userEvent.setup();

    render(
      <Tabs defaultValue="b">
        <TabsList>
          <TabsTrigger value="a">A</TabsTrigger>
          <TabsTrigger value="b">B</TabsTrigger>
        </TabsList>
        <TabsContent value="a">PanelA</TabsContent>
        <TabsContent value="b">PanelB</TabsContent>
      </Tabs>,
    );

    expect(screen.getByText("PanelB")).toBeVisible();
    expect(screen.queryByText("PanelA")).not.toBeVisible();

    await user.click(screen.getByRole("tab", { name: "A" }));

    expect(screen.getByText("PanelA")).toBeVisible();
    expect(screen.queryByText("PanelB")).not.toBeVisible();
  });
});
