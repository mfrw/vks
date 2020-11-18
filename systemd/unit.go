package systemd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/miekg/vks/pkg/system"
	"github.com/miekg/vks/pkg/unit"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func unitToPod(units map[string]*unit.UnitState) *corev1.Pod {
	if len(units) == 0 {
		return nil
	}
	name := ""
	status := ""
	var om metav1.ObjectMeta
	for k, v := range units {
		name = k
		status = v.ActiveState // need to && all the stats together.??
		om = v.Meta
		break
	}
	// order of the map is random, need to sort.
	containers := toContainers(units)
	containerStatuses := toContainerStatuses(units)

	p := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        Pod(name),
			Namespace:   om.Namespace,
			ClusterName: om.ClusterName,
			UID:         types.UID(UID(name)),
			UID:         om.UID,
			//			CreationTimestamp: // do we know?
		},
		Spec: corev1.PodSpec{
			NodeName:   system.Hostname(),
			Volumes:    []corev1.Volume{},
			Containers: containers,
		},

		// podstatus is the sum of all container statusses???
		Status: corev1.PodStatus{
			Phase:      activeStateToPhase(status),
			Conditions: activeStateToPodConditions(status, metav1.NewTime(time.Now())),
			Message:    "",
			Reason:     "",
			HostIP:     "",
			PodIP:      "127.0.0.1",
			//			StartTime:         &containerStartTime,
			ContainerStatuses: containerStatuses,
		},
	}
	return p
}

func toContainers(units map[string]*unit.UnitState) []corev1.Container {
	keys := unitNames(units)
	containers := make([]v1.Container, 0, len(units))
	for _, k := range keys {
		container := v1.Container{
			Name:      Image(k),
			Image:     Image(k),            // We not saving the image anywhere, this assume container.Name == container.Image
			Command:   []string{"/bin/sh"}, // parse from unit file?
			Resources: v1.ResourceRequirements{
				/*
					Limits: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%g", float64(c.CPU))),
						v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dM", containerMemoryMB)),
					},

					Requests: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%g", float64(c.CPU*1024/100))),
						v1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dM", containerMemoryMB)),
					},
				*/
			},
		}
		containers = append(containers, container)
	}
	return containers
}

func toContainerStatuses(units map[string]*unit.UnitState) []corev1.ContainerStatus {
	keys := unitNames(units)
	statuses := make([]v1.ContainerStatus, 0, len(units))
	for _, k := range keys {
		u := units[k]
		status := v1.ContainerStatus{
			Name:                 Name(k),
			State:                containerState(u),
			LastTerminationState: containerState(u),
			Ready:                activeStateToPhase(u.ActiveState) == v1.PodRunning,
			RestartCount:         int32(0),
			Image:                Image(k),
			ImageID:              "",
			ContainerID:          "uuid", // from name?
		}
		statuses = append(statuses, status)
	}
	return statuses
}

func containerState(u *unit.UnitState) v1.ContainerState {
	// Handle the case where the container is running.
	if u.ActiveState == "active" {
		return v1.ContainerState{
			Running: &v1.ContainerStateRunning{
				StartedAt: metav1.NewTime(time.Time(time.Now())),
			},
		}
	}

	// Handle the case where the container failed.
	if u.ActiveState == "failed" || u.ActiveState == "inactive" {
		return v1.ContainerState{
			Terminated: &v1.ContainerStateTerminated{
				ExitCode:   int32(0),
				Reason:     u.ActiveState,
				Message:    "yes",
				StartedAt:  metav1.NewTime(time.Time(time.Now())),
				FinishedAt: metav1.NewTime(time.Time(time.Now())),
			},
		}
	}

	// Handle the case where the container is pending.
	return v1.ContainerState{
		Waiting: &v1.ContainerStateWaiting{
			Reason:  u.ActiveState,
			Message: "now what",
		},
	}
}

func unitNames(units map[string]*unit.UnitState) []string {
	keys := make([]string, len(units))
	i := 0
	for k := range units {
		keys[i] = k
		i++
	}
	return sort.StringSlice(keys)
}

// objectMetaToSection converts the objectMeta into a buffer containing a [X-Kubernetes] section.
// this allows us to store any Kubernetes metadata in the unit.
func objectMetaToSection(om metav1.ObjectMeta) []byte {
	b := new(strings.Builder)
	b.WriteString("\n\n[X-Kubernetes]\n")
	fmt.Fprintf(b, "namespace=%s\n", om.Namespace)
	fmt.Fprintf(b, "clusterName=%s\n", om.ClusterName)
	fmt.Fprintf(b, "uid=%s\n", string(om.UID))
	// creation timestamp and other values? TODO
	return []byte(b.String())
}
