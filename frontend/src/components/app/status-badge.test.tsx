import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { StatusBadge } from "./status-badge";

/**
 * Tests for the status language.
 *
 * StatusBadge is used on the dashboard, the monitor list and the detail page,
 * so it is the single place that decides how status LOOKS. If it is wrong, it
 * is wrong everywhere at once.
 */

describe("StatusBadge", () => {
  it("shows readable text for every status", () => {
    // Not "renders without crashing" - that proves almost nothing. This checks
    // the badge actually SAYS something a human can read.
    const cases = [
      { status: "up", label: "Up" },
      { status: "down", label: "Down" },
      { status: "pending", label: "Pending" },
      { status: "degraded", label: "Degraded" },
    ] as const;

    for (const { status, label } of cases) {
      const { unmount } = render(<StatusBadge status={status} />);
      expect(screen.getByText(label)).toBeInTheDocument();
      unmount(); // clean up before the next render, or they stack up
    }
  });

  // THE ACCESSIBILITY TEST THAT MATTERS.
  it("never relies on colour alone to convey status", () => {
    // Roughly 1 in 12 men has some form of red/green colour blindness. A red
    // dot and a green dot are indistinguishable to them.
    //
    // So "up" and "down" must differ in their TEXT, not just their styling.
    // This is WCAG 1.4.1 (Use of Color).
    const up = render(<StatusBadge status="up" />);
    const upText = up.container.textContent;
    up.unmount();

    const down = render(<StatusBadge status="down" />);
    const downText = down.container.textContent;
    down.unmount();

    expect(upText).toBeTruthy();
    expect(downText).toBeTruthy();
    expect(upText).not.toBe(downText);
  });

  it("marks its icon as decorative", () => {
    // The icon sits next to text that already says the same thing. Without
    // aria-hidden a screen reader would announce the status twice.
    const { container } = render(<StatusBadge status="up" />);
    const icon = container.querySelector("svg");

    expect(icon).not.toBeNull();
    expect(icon).toHaveAttribute("aria-hidden", "true");
  });

  it("accepts extra classes without losing its own", () => {
    const { container } = render(
      <StatusBadge status="up" className="custom-class" />,
    );
    const badge = container.firstElementChild;

    expect(badge).toHaveClass("custom-class");
    // Its own styling must survive - a naive implementation that replaces
    // className instead of merging would drop this.
    expect(badge?.className).toContain("rounded-full");
  });
});
