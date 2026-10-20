# WCAG 2.2 success criteria

What each success criterion of the Web Content Accessibility Guidelines
(WCAG) 2.2 asks, one section per criterion, in the order WCAG numbers
them, with the criterion's number, its name and its conformance level.
These are the values the `criterion` column of the **wcag** input takes.

The numbers, names and levels are excerpts of W3C's document, copied as
they are: Web Content Accessibility Guidelines (WCAG) 2.2,
<https://www.w3.org/TR/2024/REC-WCAG22-20241212/>, status: W3C
Recommendation 12 December 2024. Copyright © 2020-2024 World Wide Web
Consortium. W3C® liability, trademark and document use rules apply.

Each summary is the METAIS project's own wording of what the criterion
asks, written for whoever fills in the table; it is not W3C's text, and
where a summary says "with stated exceptions" the criterion lists them.
The criterion's normative text is W3C's, at the address above, and it
governs: read it before deciding a conformance level. W3C has not
reviewed or endorsed this package. `sources.md` records this source and
its terms. WCAG 2.2 removed 4.1.1 Parsing, which is therefore not listed.

## 1.1.1 Non-text Content (Level A)

Every image, control or other non-text item that conveys something has a text equivalent that conveys the same, and purely decorative items are hidden from assistive technology.

## 1.2.1 Audio-only and Video-only (Prerecorded) (Level A)

A recorded clip that is sound alone has a transcript, and one that is picture alone has a transcript or a soundtrack that describes it.

## 1.2.2 Captions (Prerecorded) (Level A)

Recorded video with sound carries captions of its speech and meaningful sounds.

## 1.2.3 Audio Description or Media Alternative (Prerecorded) (Level A)

Recorded video has either a narration of what is seen or a full text alternative.

## 1.2.4 Captions (Live) (Level AA)

Live video with sound carries captions as it plays.

## 1.2.5 Audio Description (Prerecorded) (Level AA)

Recorded video has a narration of the visual information the soundtrack leaves out.

## 1.2.6 Sign Language (Prerecorded) (Level AAA)

Recorded video with sound has a sign language interpretation of its sound.

## 1.2.7 Extended Audio Description (Prerecorded) (Level AAA)

Where the pauses in a recorded video are too short to describe it, the video stops to let the description finish.

## 1.2.8 Media Alternative (Prerecorded) (Level AAA)

Every recorded video has a full text alternative.

## 1.2.9 Audio-only (Live) (Level AAA)

Live sound with no picture has a text alternative that keeps pace with it.

## 1.3.1 Info and Relationships (Level A)

Structure the layout shows, such as headings, lists, tables and form groupings, is also available to software.

## 1.3.2 Meaningful Sequence (Level A)

Where the order of the content changes what it means, software can determine that order.

## 1.3.3 Sensory Characteristics (Level A)

Instructions do not depend on shape, colour, size, position or sound alone.

## 1.3.4 Orientation (Level AA)

Content works in portrait and in landscape unless one orientation is essential.

## 1.3.5 Identify Input Purpose (Level AA)

Form fields that collect information about the user say, in a way software can read, what each field is for.

## 1.3.6 Identify Purpose (Level AAA)

Software can determine the purpose of interface components, icons and regions.

## 1.4.1 Use of Color (Level A)

Colour is never the only way information, an action or a state is shown.

## 1.4.2 Audio Control (Level A)

Sound that plays by itself for more than a few seconds can be paused, stopped or turned down separately from the system volume.

## 1.4.3 Contrast (Minimum) (Level AA)

Text stands out from its background at a contrast ratio of at least 4.5 to 1, or 3 to 1 for large text, with stated exceptions.

## 1.4.4 Resize Text (Level AA)

Text can be enlarged to twice its size without assistive technology and without losing content or function.

## 1.4.5 Images of Text (Level AA)

Text is shown as text rather than as a picture of text, unless the picture is essential or can be customized.

## 1.4.6 Contrast (Enhanced) (Level AAA)

Text stands out from its background at a contrast ratio of at least 7 to 1, or 4.5 to 1 for large text.

## 1.4.7 Low or No Background Audio (Level AAA)

Recorded speech has no background sound, background sound that can be switched off, or background sound well below the speech.

## 1.4.8 Visual Presentation (Level AAA)

Blocks of text can be shown with the user's colours, a limited line length, unjustified, with generous spacing, and resized without horizontal scrolling.

## 1.4.9 Images of Text (No Exception) (Level AAA)

Pictures of text are used only for decoration or where the particular presentation is essential.

## 1.4.10 Reflow (Level AA)

Content fits a narrow viewport without scrolling in two directions, except where a two-dimensional layout is essential.

## 1.4.11 Non-text Contrast (Level AA)

The parts of controls and of graphics needed to understand them stand out from adjacent colours at a ratio of at least 3 to 1.

## 1.4.12 Text Spacing (Level AA)

Nothing is lost when a user increases the spacing of lines, paragraphs, letters and words to stated amounts.

## 1.4.13 Content on Hover or Focus (Level AA)

Content that appears on hover or focus can be dismissed, can itself be hovered, and stays until the user moves on.

## 2.1.1 Keyboard (Level A)

Everything can be done from a keyboard, without timing individual keystrokes, unless the task depends on the path of the movement.

## 2.1.2 No Keyboard Trap (Level A)

Keyboard focus that can move into a component can move out of it again, and the user is told how if it takes more than the usual keys.

## 2.1.3 Keyboard (No Exception) (Level AAA)

Everything can be done from a keyboard, with no exception for path-dependent input.

## 2.1.4 Character Key Shortcuts (Level A)

Single-character keyboard shortcuts can be turned off, remapped, or work only while their component has focus.

## 2.2.1 Timing Adjustable (Level A)

Each time limit can be turned off, adjusted or extended, with stated exceptions.

## 2.2.2 Pause, Stop, Hide (Level A)

Moving, blinking, scrolling or automatically updating content can be paused, stopped or hidden.

## 2.2.3 No Timing (Level AAA)

Timing is not part of the task, except for live events and real-time media.

## 2.2.4 Interruptions (Level AAA)

Interruptions can be postponed or suppressed, except in emergencies.

## 2.2.5 Re-authenticating (Level AAA)

When a session expires, the user can sign in again and carry on without losing data.

## 2.2.6 Timeouts (Level AAA)

Users are warned how long they can be inactive before data is lost, unless data is kept for more than twenty hours.

## 2.3.1 Three Flashes or Below Threshold (Level A)

Nothing flashes more than three times in any one second, or the flashing stays below the general and red flash thresholds.

## 2.3.2 Three Flashes (Level AAA)

Nothing flashes more than three times in any one second.

## 2.3.3 Animation from Interactions (Level AAA)

Motion animation set off by an interaction can be turned off, unless it is essential.

## 2.4.1 Bypass Blocks (Level A)

There is a way to skip blocks of content that repeat on several pages.

## 2.4.2 Page Titled (Level A)

Each page has a title that says what it is about or for.

## 2.4.3 Focus Order (Level A)

Keyboard focus moves through the content in an order that keeps its meaning and operation.

## 2.4.4 Link Purpose (In Context) (Level A)

What a link does can be told from its text, or from its text with the context software can determine.

## 2.4.5 Multiple Ways (Level AA)

There is more than one way to find a page within a set of pages, unless the page is a step in a process.

## 2.4.6 Headings and Labels (Level AA)

Headings and labels say what their topic or purpose is.

## 2.4.7 Focus Visible (Level AA)

Whatever the keyboard focus is on is visibly marked.

## 2.4.8 Location (Level AAA)

Users can tell where they are within a set of pages.

## 2.4.9 Link Purpose (Link Only) (Level AAA)

What a link does can be told from its text alone.

## 2.4.10 Section Headings (Level AAA)

Sections of content are organized under headings.

## 2.4.11 Focus Not Obscured (Minimum) (Level AA)

A component that receives keyboard focus is not entirely hidden by content the author placed over it.

## 2.4.12 Focus Not Obscured (Enhanced) (Level AAA)

No part of a component that receives keyboard focus is hidden by content the author placed over it.

## 2.4.13 Focus Appearance (Level AAA)

The focus indicator is large enough and contrasts enough with its unfocused state to be seen easily.

## 2.5.1 Pointer Gestures (Level A)

Anything done with a gesture that has several points of contact or follows a path can also be done with a single pointer and no path.

## 2.5.2 Pointer Cancellation (Level A)

Actions run on release of a pointer rather than on press, or can be aborted or undone.

## 2.5.3 Label in Name (Level A)

The name software reads for a labelled control contains the text of its visible label.

## 2.5.4 Motion Actuation (Level A)

Anything done by moving or shaking the device can also be done through the interface, and the motion response can be turned off.

## 2.5.5 Target Size (Enhanced) (Level AAA)

Pointer targets are at least 44 by 44 CSS pixels, with stated exceptions.

## 2.5.6 Concurrent Input Mechanisms (Level AAA)

Content does not restrict the user to one input modality among those the platform offers.

## 2.5.7 Dragging Movements (Level AA)

Anything done by dragging can also be done with a single pointer without dragging, unless dragging is essential.

## 2.5.8 Target Size (Minimum) (Level AA)

Pointer targets are at least 24 by 24 CSS pixels, or spaced so that they do not crowd one another, with stated exceptions.

## 3.1.1 Language of Page (Level A)

Software can determine the main human language of each page.

## 3.1.2 Language of Parts (Level AA)

Software can determine the language of each passage or phrase that differs from the page's.

## 3.1.3 Unusual Words (Level AAA)

There is a way to find the meaning of words used in an unusual or restricted way, idioms and jargon included.

## 3.1.4 Abbreviations (Level AAA)

There is a way to find the expanded form or meaning of an abbreviation.

## 3.1.5 Reading Level (Level AAA)

Text that needs more than lower secondary reading ability has a simpler version or supplementary content.

## 3.1.6 Pronunciation (Level AAA)

There is a way to find the pronunciation of a word whose meaning depends on it.

## 3.2.1 On Focus (Level A)

Moving focus to a component does not by itself change the context.

## 3.2.2 On Input (Level A)

Changing a setting does not change the context unless the user was told beforehand.

## 3.2.3 Consistent Navigation (Level AA)

Navigation repeated across pages appears in the same relative order each time.

## 3.2.4 Consistent Identification (Level AA)

Components with the same function across pages are identified in the same way.

## 3.2.5 Change on Request (Level AAA)

Changes of context happen only when the user asks, or can be turned off.

## 3.2.6 Consistent Help (Level A)

Help mechanisms repeated across pages appear in the same relative order each time.

## 3.3.1 Error Identification (Level A)

An input error that is detected automatically is pointed out and described to the user in text.

## 3.3.2 Labels or Instructions (Level A)

Fields that need input have labels or instructions.

## 3.3.3 Error Suggestion (Level AA)

When an input error is detected and a correction is known, the correction is suggested, unless that would undermine security or purpose.

## 3.3.4 Error Prevention (Legal, Financial, Data) (Level AA)

Submissions with legal or financial effect, or that change stored data, can be reversed, are checked, or can be reviewed and confirmed.

## 3.3.5 Help (Level AAA)

Help that depends on the context is available.

## 3.3.6 Error Prevention (All) (Level AAA)

Every submission of information can be reversed, is checked, or can be reviewed and confirmed.

## 3.3.7 Redundant Entry (Level A)

Information the user already entered in a process is filled in or offered again rather than asked for twice, with stated exceptions.

## 3.3.8 Accessible Authentication (Minimum) (Level AA)

Signing in does not depend on a cognitive function test, such as remembering or transcribing, unless an alternative or assistance is offered.

## 3.3.9 Accessible Authentication (Enhanced) (Level AAA)

Signing in does not depend on a cognitive function test, recognising objects and personal content included, unless an alternative or assistance is offered.

## 4.1.2 Name, Role, Value (Level A)

Software can determine the name and role of every interface component, and read and be told of changes to its states, properties and values.

## 4.1.3 Status Messages (Level AA)

Status messages can be announced by assistive technology without receiving focus.
