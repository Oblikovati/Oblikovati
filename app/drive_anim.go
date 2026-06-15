// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/math"
	"oblikovati.org/model/assembly"
	"oblikovati.org/model/occurrence"
)

// Joint drive playback (M12-F03, Oblikovati/Oblikovati#366): the Drive command precomputes a
// joint's motion frames (assembly.DriveJoint) and the head plays them back, applying each
// frame's occurrence placements so the assembly animates through the joint's range of motion.
// Playback loops while active; stopping restores the pre-drive rest pose (the drive itself is
// non-destructive). The cadence/looping logic is pure so it is unit-testable.

// driveFrameSeconds is how long each motion frame is shown — the playback cadence.
const driveFrameSeconds = 0.04

// driveAnimation is an in-progress drive playback: the motion frames, the occurrences they
// move, the rest pose to restore on stop, and the playback cursor.
type driveAnimation struct {
	frames    []assembly.DriveFrame
	container *occurrence.Occurrences
	occs      map[uint64]*occurrence.Occurrence
	rest      map[uint64]math.Matrix4
	elapsed   float64
	active    bool
}

// StartDriveAnimation computes the joint's motion frames and begins playback. It errors when
// there is no active assembly or the joint is not drivable; a drive with no frames is a no-op.
func (s *Session) StartDriveAnimation(jointID uint64, settings assembly.DriveSettings) error {
	asm, err := activeAssembly(s)
	if err != nil {
		return err
	}
	res, err := asm.DriveJoint(jointID, settings)
	if err != nil {
		return err
	}
	if len(res.Frames) == 0 {
		return nil
	}
	occs, rest := map[uint64]*occurrence.Occurrence{}, map[uint64]math.Matrix4{}
	for _, o := range asm.Occurrences().All() {
		occs[o.ID()], rest[o.ID()] = o, o.Transform()
	}
	s.driveAnim = driveAnimation{frames: res.Frames, container: asm.Occurrences(), occs: occs, rest: rest, active: true}
	return nil
}

// DriveAnimating reports whether a drive is playing, so the head ticks it each frame.
func (s *Session) DriveAnimating() bool { return s.driveAnim.active }

// TickDriveAnimation advances playback by dt seconds, applying the current motion frame
// (looping through the frames). It is a no-op when no drive is active.
func (s *Session) TickDriveAnimation(dt float64) {
	if !s.driveAnim.active || dt < 0 {
		return
	}
	s.driveAnim.elapsed += dt
	idx := int(s.driveAnim.elapsed/driveFrameSeconds) % len(s.driveAnim.frames)
	s.applyDrivePlacements(s.driveAnim.frames[idx].Placements)
}

// StopDriveAnimation ends playback and restores the assembly's pre-drive rest pose.
func (s *Session) StopDriveAnimation() {
	if !s.driveAnim.active {
		return
	}
	s.driveAnim.active = false
	s.driveAnim.container.SuspendNotifications()
	defer s.driveAnim.container.ResumeNotifications()
	for id, m := range s.driveAnim.rest {
		if o := s.driveAnim.occs[id]; o != nil {
			o.SetTransform(m)
		}
	}
}

// applyDrivePlacements writes one frame's occurrence transforms in a single notification batch.
func (s *Session) applyDrivePlacements(placements []assembly.OccurrencePlacement) {
	s.driveAnim.container.SuspendNotifications()
	defer s.driveAnim.container.ResumeNotifications()
	for _, p := range placements {
		if o := s.driveAnim.occs[p.Occurrence]; o != nil {
			o.SetTransform(p.Transform)
		}
	}
}
