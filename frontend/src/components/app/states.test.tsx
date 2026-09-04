import { render, screen } from "@testing-library/react";
import { Radio } from "lucide-react";
import { describe, expect, it } from "vitest";

import { EmptyState, ErrorState, Skeleton } from "./states";

/**
 * Tests for the loading / empty / error states.
 *
 * Every component that reads server data must handle four situations:
 * loading, success, empty and error. Only the success path tends to get
 * attention, yet the other three are what users hit on slow connections, new
 * accounts, and bad days for the server.
 *
 * These tests pin down the RULES those states must follow, not just that they
 * render something.
 */

describe("EmptyState", () => {
  it("explains what is empty and why, not just that it is empty", () => {
    render(
      <EmptyState
        icon={Radio}
        title="No monitors yet"
        description="Monitors check your endpoints on a schedule."
      />,
    );

    // A blank area with no explanation reads as a broken page. Both halves
    // must be present: what this is, and why there is nothing here.
    expect(screen.getByText("No monitors yet")).toBeInTheDocument();
    expect(
      screen.getByText("Monitors check your endpoints on a schedule."),
    ).toBeInTheDocument();
  });

  it("can offer one action to fix the emptiness", () => {
    render(
      <EmptyState
        title="No monitors yet"
        description="Nothing here."
        action={<button>Create monitor</button>}
      />,
    );

    // An empty state that tells you nothing is missing an opportunity: the
    // user is right there, and the next step is obvious to us but not to them.
    expect(
      screen.getByRole("button", { name: "Create monitor" }),
    ).toBeInTheDocument();
  });

  it("works without an icon or action", () => {
    // Both are optional, and forgetting them must not crash the page.
    render(<EmptyState title="Nothing" description="Nothing at all." />);
    expect(screen.getByText("Nothing")).toBeInTheDocument();
  });
});

describe("ErrorState", () => {
  it("announces itself to screen readers", () => {
    render(<ErrorState />);

    // role="alert" makes assistive technology speak immediately. Without it a
    // blind user sits in silence wondering why nothing loaded - the visual
    // red box conveys nothing to them.
    expect(screen.getByRole("alert")).toBeInTheDocument();
  });

  it("always offers a way to recover", () => {
    render(
      <ErrorState
        title="Could not load your monitors"
        description="The server did not respond."
        onRetry={<button>Try again</button>}
      />,
    );

    expect(screen.getByText("Could not load your monitors")).toBeInTheDocument();
    // An error with no way forward leaves the user stuck reloading the page
    // and hoping.
    expect(
      screen.getByRole("button", { name: "Try again" }),
    ).toBeInTheDocument();
  });

  it("has a sensible default message", () => {
    // Even when a caller passes nothing, the user must see words rather than
    // an empty red box.
    render(<ErrorState />);
    expect(screen.getByText(/could not load/i)).toBeInTheDocument();
  });
});

describe("Skeleton", () => {
  it("is hidden from screen readers", () => {
    const { container } = render(<Skeleton className="h-4 w-20" />);
    const element = container.firstElementChild;

    // A skeleton is a visual placeholder with no meaning. Announcing it would
    // just add noise between "loading" and the actual content.
    expect(element).toHaveAttribute("aria-hidden", "true");
  });

  it("accepts sizing classes", () => {
    // Skeletons MUST be sized to match the real content. A skeleton with the
    // wrong dimensions causes the very layout jump it exists to prevent -
    // which is worse than showing nothing at all.
    const { container } = render(<Skeleton className="h-7 w-16" />);
    expect(container.firstElementChild).toHaveClass("h-7", "w-16");
  });
});
