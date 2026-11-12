# Nystagmus Training

A macOS status bar app that asks you to follow a slow green dot with your eyes for a few minutes after every couple of hours at the computer.

Following a moving target is called smooth pursuit. People with nystagmus (involuntary, rhythmic eye movements) often track poorly, so the dot moves slowly: at most 8 degrees of visual angle per second on a screen 60 cm away, along a curve with no sharp turns.

It is for people with nystagmus who want regular pursuit practice built into their computer day. It is an experiment, not a treatment. No clinical trial has shown that following a dot reduces nystagmus or sharpens vision. Ask your eye specialist what suits your type of nystagmus.

## Install

Requires macOS 12 or later.

```sh
curl -L -o NystagmusTraining.zip https://github.com/lovlar-ab/nystagmus-training/releases/latest/download/NystagmusTraining.zip
unzip NystagmusTraining.zip -d /Applications
open "/Applications/Nystagmus Training.app"
```

If you download the zip in a browser instead, macOS blocks the unsigned app until you run:

```sh
xattr -dr com.apple.quarantine "/Applications/Nystagmus Training.app"
```

To build it yourself you need Go 1.26. Run `./build.sh` on macOS or Linux and the app lands in `dist/`.

## Use

Click the eye icon in the status bar.

- The top line shows how much computer use is left before the next session. Time counts only while you type or move the mouse.
- "Start session" starts one now. A session fills the screen with a dark backdrop and the dot, which takes a new random path each time.
- Esc stops a session.
- The up and down arrow keys make the backdrop darker or more see-through during a session.
- The "Active" slider has two handles. Drag them to set the hours of the day when sessions may start on their own, for example 08:00 to 18:00.
- The "Start after" slider sets how much use triggers a session, from 15 minutes to 4 hours.
- The "Session length" slider sets how long the dot runs, from 1 to 15 minutes.
- "Open at login" starts the app when you log in.

## Sources

- [Efficacy of perceptual learning among patients with infantile nystagmus: a randomised controlled trial](https://pubmed.ncbi.nlm.nih.gov/41402029/) (2025). The training with trial evidence is a contrast detection task, not dot following. This is why the app claims no treatment effect.
- [Kitamura et al., Frontiers in Neurology](https://www.frontiersin.org/journals/neurology/articles/10.3389/fneur.2018.00622/full) (2018). Smooth pursuit of targets moving at up to 20 degrees per second was impaired in people with infantile nystagmus. The dot therefore peaks at 8 degrees per second and moves along a smooth curve.
- [Symptomatic treatment of infantile nystagmus: a systematic review](https://www.ncbi.nlm.nih.gov/pmc/articles/PMC12222088/). Checked for existing evidence on eye movement training.
- [The effects of breaks on digital eye strain: testing the 20-20-20 rule](https://pubmed.ncbi.nlm.nih.gov/35963776/) and [Review of Optometry on the same question](https://www.reviewofoptometry.com/news/article/202020-still-not-enough-to-alleviate-eye-strain). Evidence for very short breaks is mixed, and a 5 minute break may work better. The default session is 5 minutes.
